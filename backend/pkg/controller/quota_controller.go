/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	aijobapi "k8s.io/ai-task-controller/pkg/apis/aijob/v1alpha1"

	joblister "k8s.io/ai-task-controller/pkg/generated/aijob/listers/aijob/v1alpha1"
	quotascheme "k8s.io/ai-task-controller/pkg/generated/tenantquota/clientset/versioned/scheme"
	quotaclientest "k8s.io/ai-task-controller/pkg/generated/tenantquota/clientset/versioned/typed/tenantquota/v1alpha1"
	informers "k8s.io/ai-task-controller/pkg/generated/tenantquota/informers/externalversions/tenantquota/v1alpha1"
	quotalister "k8s.io/ai-task-controller/pkg/generated/tenantquota/listers/tenantquota/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a Tenantquota is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Tenantquota fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Tenantquota"
	// MessageResourceSynced is the message used for an Event fired when Tenantquota
	// is synced successfully
	MessageResourceSynced = "Quota synced successfully"
)

// QuotaController is the controller implementation for Tenantquota resources
type QuotaController struct {
	kubeclientset kubernetes.Interface
	jobLister     joblister.AIJobLister
	quotaLister   quotalister.TenantQuotaLister
	quotaClient   *quotaclientest.AisystemV1alpha1Client
	quotaSynced   cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewQuotaController returns a new quota controller
func NewQuotaController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	quotaInformer informers.TenantQuotaInformer,
	quotaClient *quotaclientest.AisystemV1alpha1Client,
) *QuotaController {

	// Create event broadcaster
	// Add quota-controller types to the default Kubernetes Scheme so Events can be
	// logged for quota-controller types.
	utilruntime.Must(quotascheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "quota-controller"})

	controller := &QuotaController{
		kubeclientset: kubeclientset,
		quotaLister:   quotaInformer.Lister(),
		quotaSynced:   quotaInformer.Informer().HasSynced,
		quotaClient:   quotaClient,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Tenantquotas"),
		recorder:      recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Quota resources change
	quotaInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueQuota,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueQuota(new)
		},
		DeleteFunc: controller.enqueueQuota,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *QuotaController) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Quota controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.quotaSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Quota resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *QuotaController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *QuotaController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Quota resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler syncs the TenantQuota with QuotaInfosData
func (c *QuotaController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Quota resource with this namespace/name
	tq, err := c.quotaLister.TenantQuotas(namespace).Get(name)
	if err != nil {
		// The Quota resource may no longer exist, delete in the QuotaInfosData store
		if errors.IsNotFound(err) {
			QuotaInfosData.DeleteQuotaInfo(key)
			return nil
		}
		return err
	}
	// if quotainfo is added, should adds the aijobs in the quotainfo
	added, quotaInfo := QuotaInfosData.AddOrUpdateQuotaInfo(key, tq)
	if added {
		jobList, err := c.jobLister.AIJobs(tq.Namespace).List(labels.Everything())
		if err != nil {
			return err
		}
		for _, job := range jobList {
			// only add Running job
			// todo:
			if job.Status.Phase == aijobapi.Running {
				quotaInfo.AddJob(job)
			}
		}
	}

	c.recorder.Event(tq, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// enqueueQuota takes a Quota resource and converts it into a namespace/name
// string which is then put onto the work queue.
func (c *QuotaController) enqueueQuota(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// todo: update TenantQuota Used Status from QuotaInfosData
func (c *QuotaController) updateQuotaStatus(quotaInfo *QuotaInfo) error {
	quota, err := c.quotaLister.TenantQuotas(quotaInfo.Namespace).Get(quotaInfo.Name)
	if err != nil {
		return err
	}
	quota.Status.Hard = quotaInfo.HardUsed.DeepCopy()
	quota.Status.Soft = quotaInfo.SoftUsed.DeepCopy()
	_, err = c.quotaClient.TenantQuotas(quota.Namespace).UpdateStatus(context.TODO(), quota, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
