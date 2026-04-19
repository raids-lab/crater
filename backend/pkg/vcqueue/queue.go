package vcqueue

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/constants"
)

var queueLocks sync.Map // map[queueName]*sync.Mutex

func getQueueLock(queueName string) *sync.Mutex {
	lock, _ := queueLocks.LoadOrStore(queueName, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// checkQueueExists checks if user queue exists without creating it
func checkQueueExists(ctx context.Context, cli client.Client, name string) (bool, error) {
	namespace := config.GetConfig().Namespaces.Job

	queue := &scheduling.Queue{}
	err := cli.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, queue)
	if err == nil {
		return true, nil
	}
	if errors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

// EnsureUserQueueExists ensures user queue exists, creating it if necessary
func EnsureUserQueueExists(ctx context.Context, cli client.Client, token util.JWTMessage, accountID, userID uint) error {
	if accountID == model.DefaultAccountID {
		return nil
	}
	queueName := GetUserQueueName(accountID, userID)
	accountQueueName := GetAccountLogicQueueName(accountID)
	lock := getQueueLock(queueName)
	lock.Lock()
	defer lock.Unlock()

	exists, err := checkQueueExists(ctx, cli, queueName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	uq := query.UserAccount
	userAccount, err := uq.WithContext(ctx).Where(uq.UserID.Eq(userID), uq.AccountID.Eq(accountID)).First()
	if err != nil {
		return fmt.Errorf("user %d is not in account %d", userID, accountID)
	}

	quota := userAccount.Quota.Data()
	if err := CreateQueue(ctx, cli, token, queueName, &accountQueueName, &quota); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	klog.Infof("Created user queue %s for user %d in account %d", queueName, userID, accountID)
	return nil
}

// EnsureAccountQueueExists ensures account queue exists, creating it if necessary
func EnsureAccountQueueExists(ctx context.Context, cli client.Client, token util.JWTMessage, accountID uint) error {
	if accountID == model.DefaultAccountID {
		return nil
	}
	queueName := GetAccountLogicQueueName(accountID)

	lock := getQueueLock(queueName)
	lock.Lock()
	defer lock.Unlock()

	exists, err := checkQueueExists(ctx, cli, queueName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	account := query.Account
	acc, err := account.WithContext(ctx).Where(account.ID.Eq(accountID)).First()
	if err != nil {
		return fmt.Errorf("account %d not found", accountID)
	}

	quota := acc.Quota.Data()
	if err := CreateQueue(ctx, cli, token, queueName, nil, &quota); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	klog.Infof("Created account queue %s for account %d", queueName, accountID)
	return nil
}

func UpdateQueue(ctx context.Context, cli client.Client, queueName string, quota model.QueueQuota) error {
	namespace := config.GetConfig().Namespaces.Job
	lock := getQueueLock(queueName)
	lock.Lock()
	defer lock.Unlock()

	vcQueue := &scheduling.Queue{}
	if err := cli.Get(ctx, client.ObjectKey{Name: queueName, Namespace: namespace}, vcQueue); err != nil {
		return err
	}

	vcQueue.Spec.Guarantee = scheduling.Guarantee{Resource: quota.Guaranteed}
	vcQueue.Spec.Deserved = quota.Deserved
	vcQueue.Spec.Capability = quota.Capability

	if err := cli.Update(ctx, vcQueue); err != nil {
		return err
	}
	return nil
}

func CreateQueue(
	ctx context.Context,
	cli client.Client,
	token util.JWTMessage,
	queueName string,
	parentName *string,
	quota *model.QueueQuota,
) error {
	namespace := config.GetConfig().Namespaces.Job
	labels := map[string]string{
		constants.LabelKeyQueueCreatedBy: token.Username,
	}

	volcanoQueue := &scheduling.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      queueName,
			Namespace: namespace,
			Labels:    labels,
		},
	}
	if parentName != nil {
		volcanoQueue.Spec.Parent = *parentName
	}
	if quota != nil {
		volcanoQueue.Spec.Guarantee = scheduling.Guarantee{Resource: quota.Guaranteed}
		volcanoQueue.Spec.Capability = quota.Capability
		volcanoQueue.Spec.Deserved = quota.Deserved
	}

	if err := cli.Create(ctx, volcanoQueue); err != nil {
		return err
	}
	return nil
}

func DeleteQueue(ctx context.Context, cli client.Client, queueName string) error {
	namespace := config.GetConfig().Namespaces.Job

	lock := getQueueLock(queueName)
	lock.Lock()
	defer lock.Unlock()

	queue := &scheduling.Queue{}
	if err := cli.Get(ctx, client.ObjectKey{Name: queueName, Namespace: namespace}, queue); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Queue %s not found, skipping deletion", queueName)
			return nil
		}
		return err
	}

	if queue.Status.Running != 0 {
		return fmt.Errorf("queue still have running pod")
	}

	if err := cli.Delete(ctx, queue); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Queue %s was already deleted", queueName)
			return nil
		}
		return err
	}
	return nil
}

func GetUserQueueName(accountID, userID uint) string {
	return fmt.Sprintf("q-a%d-u%d", accountID, userID)
}

func GetAccountQueueName(accountName string) string {
	return accountName
}

func GetAccountLogicQueueName(accountID uint) string {
	return fmt.Sprintf("q-a%d", accountID)
}
