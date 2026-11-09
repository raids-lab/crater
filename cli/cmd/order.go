package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	orderTypes          = []string{"job", "dataset"}
	orderEditableStatus = []string{"Pending"}
	orderListStatuses   = []string{"Pending", "Approved", "Rejected", "Canceled"}
)

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "Manage approval orders",
	Long:  "Submit, edit, cancel, and view approval orders.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var orderLsCmd = &cobra.Command{Use: "ls", Short: "List approval orders you created", Args: noArgs, RunE: runOrderLs}
var orderGetCmd = &cobra.Command{Use: "get <id>", Short: "View an approval order you created", Args: exactArgs(1, "id"), RunE: runOrderGet}
var orderByNameCmd = &cobra.Command{Use: "by-name <name>", Short: "List approval orders by name", Args: exactArgs(1, "order-name"), RunE: runOrderByName}
var orderSubmitCmd = &cobra.Command{Use: "submit", Short: "Submit an approval order", Args: noArgs, RunE: runOrderSubmit}
var orderEditCmd = &cobra.Command{Use: "edit <id>", Short: "Edit a pending approval order", Args: exactArgs(1, "id"), RunE: runOrderEdit}
var orderCancelCmd = &cobra.Command{Use: "cancel <id>", Short: "Cancel an approval order", Args: exactArgs(1, "id"), RunE: runOrderCancel}
var adminOrderCmd = &cobra.Command{Use: "order", Short: "Manage admin approval orders"}
var adminOrderLsCmd = &cobra.Command{Use: "ls", Short: "List approval orders", Args: noArgs, RunE: runAdminOrderLs}
var adminOrderGetCmd = &cobra.Command{Use: "get <id>", Short: "Get an approval order", Args: exactArgs(1, "id"), RunE: runAdminOrderGet}
var adminOrderApproveCmd = &cobra.Command{Use: "approve <id>", Short: "Approve an approval order", Args: exactArgs(1, "id"), RunE: runAdminOrderApprove}
var adminOrderRejectCmd = &cobra.Command{Use: "reject <id>", Short: "Reject an approval order", Args: exactArgs(1, "id"), RunE: runAdminOrderReject}
var adminOrderCheckCmd = &cobra.Command{Use: "check", Short: "Cancel invalid pending job approval orders", Args: noArgs, RunE: runAdminOrderCheck}

func runOrderLs(cmd *cobra.Command, _ []string) error {
	return listApprovalOrders(cmd, false)
}

func runAdminOrderLs(cmd *cobra.Command, _ []string) error {
	return listApprovalOrders(cmd, true)
}

type orderListOptions struct {
	api.ListOptions
	Admin   bool
	Status  string
	Type    string
	Creator string
	Search  string
}

func listApprovalOrders(cmd *cobra.Command, admin bool) error {
	options, err := readOrderListOptions(cmd, admin)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	orders, err := client.ListApprovalOrders(admin)
	if err != nil {
		return cliErrFromAPI(err)
	}
	orders = filterApprovalOrders(orders, options)
	sortApprovalOrders(orders)
	page := paginateList(orders, options.ListOptions)
	return writeListPage("orders", page, options.AllPages, printOrderTable)
}

func readOrderListOptions(cmd *cobra.Command, admin bool) (orderListOptions, error) {
	listOptions, issues := listPaginationOptions(cmd, maxCLIPageSize)
	status, _ := cmd.Flags().GetString("status")
	orderType, _ := cmd.Flags().GetString("type")
	search, _ := cmd.Flags().GetString("search")
	status = strings.TrimSpace(status)
	orderType = strings.TrimSpace(orderType)
	search = strings.TrimSpace(search)

	creator := ""
	if admin {
		creator, _ = cmd.Flags().GetString("creator")
		creator = strings.TrimSpace(creator)
	}
	if status != "" && !slices.Contains(orderListStatuses, status) {
		issues = append(issues, invalidIssue("status", i18n.T("err_invalid_status", status)))
	}
	if orderType != "" && !slices.Contains(orderTypes, orderType) {
		issues = append(issues, invalidIssue("type", i18n.T("err_invalid_order_type", orderType)))
	}
	if len(issues) > 0 {
		return orderListOptions{}, errUsageFromIssues(issues)
	}
	return orderListOptions{
		ListOptions: listOptions,
		Admin:       admin,
		Status:      status,
		Type:        orderType,
		Creator:     creator,
		Search:      search,
	}, nil
}

func filterApprovalOrders(
	orders []api.ApprovalOrder,
	options orderListOptions,
) []api.ApprovalOrder {
	creator := strings.ToLower(options.Creator)
	search := strings.ToLower(options.Search)
	filtered := make([]api.ApprovalOrder, 0, len(orders))
	for _, order := range orders {
		if options.Status != "" && order.Status != options.Status {
			continue
		}
		if options.Type != "" && order.Type != options.Type {
			continue
		}
		creatorUsername := strings.ToLower(order.Creator.Username)
		creatorNickname := strings.ToLower(order.Creator.Nickname)
		if options.Admin && creator != "" &&
			!strings.Contains(creatorUsername, creator) &&
			!strings.Contains(creatorNickname, creator) {
			continue
		}
		if search != "" &&
			!strings.Contains(strings.ToLower(order.Name), search) &&
			!strings.Contains(creatorUsername, search) &&
			!strings.Contains(creatorNickname, search) {
			continue
		}
		filtered = append(filtered, order)
	}
	return filtered
}

func sortApprovalOrders(orders []api.ApprovalOrder) {
	statusRank := map[string]int{
		"Pending": 0, "Approved": 1, "Rejected": 2, "Canceled": 3,
	}
	sort.SliceStable(orders, func(left, right int) bool {
		leftStatus, ok := statusRank[orders[left].Status]
		if !ok {
			leftStatus = len(statusRank)
		}
		rightStatus, ok := statusRank[orders[right].Status]
		if !ok {
			rightStatus = len(statusRank)
		}
		if leftStatus != rightStatus {
			return leftStatus < rightStatus
		}
		if orders[left].Status != orders[right].Status {
			return orders[left].Status < orders[right].Status
		}
		if !orders[left].CreatedAt.Equal(orders[right].CreatedAt) {
			return orders[left].CreatedAt.After(orders[right].CreatedAt)
		}
		return orders[left].ID > orders[right].ID
	})
}

func runOrderGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	err = runRawRead(cmd, rawReadSpec{PayloadKey: "order", Path: fmt.Sprintf("%s/%s", api.ApprovalOrderPrefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
	var cliErr *clierror.Error
	if errors.As(err, &cliErr) && cliErr.Code == errorcodes.ErrForbidden403 &&
		cliErr.Context["crater_code"] == api.CodePermissionDenied {
		if st, loadErr := session.LoadState(); loadErr == nil {
			if info, ok := session.ActiveAuthInfo(st); ok && info.Role == "admin" {
				suggestedCommand := "crater admin order get " + api.UintPath(id)
				cliErr.Context["suggested_command"] = suggestedCommand
				cliErr.Message += "\n" + i18n.T("order_admin_get_hint", suggestedCommand)
			}
		}
	}
	return err
}

func runAdminOrderGet(_ *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	order, err := client.GetApprovalOrder(id, true)
	if err != nil {
		return cliErrFromAPI(err)
	}

	payload := map[string]interface{}{"order": order}
	var job *api.JobDetail
	var jobErr *clierror.Error
	if order.Type == "job" {
		job, err = client.GetJob(order.Name)
		if err != nil {
			jobErr = cliErrFromAPI(err)
			payload["job_error"] = map[string]string{"code": jobErr.Code, "message": jobErr.Message}
		} else {
			payload["job"] = map[string]interface{}{"jobName": job.JobName, "status": job.Status, "queue": job.Queue, "resources": job.Resources}
		}
	}
	var history *approvalOrderHistory
	var historyErr string
	if order.Type == "job" {
		history, historyErr = readJobApprovalHistory(client, order)
		if historyErr == "" {
			payload["history"] = history
		} else {
			payload["history_error"] = historyErr
		}
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(payload))
	}
	printAdminOrderDetail(order, job, jobErr)
	if history != nil {
		printJobApprovalHistory(history)
	} else if historyErr != "" {
		fmt.Printf("\n%s\n", i18n.T("order_history_unavailable", historyErr))
	}
	return nil
}

func printAdminOrderDetail(order *api.ApprovalOrder, job *api.JobDetail, jobErr *clierror.Error) {
	fmt.Printf("%s: %d\n", i18n.T("table_id"), order.ID)
	fmt.Printf("%s: %s\n", i18n.T("table_name"), order.Name)
	fmt.Printf("%s: %s\n", i18n.T("table_type"), order.Type)
	fmt.Printf("%s: %s\n", i18n.T("table_status"), order.Status)
	fmt.Printf("%s: %s\n", i18n.T("order_label_creator"), emptyDash(order.Creator.Username))
	fmt.Printf("%s: %s\n", i18n.T("table_created_at"), formatAPITime(order.CreatedAt))
	fmt.Printf("%s: %s\n", i18n.T("order_label_reason"), emptyDash(order.Content.ApprovalOrderReason))
	if order.Type == "job" {
		fmt.Printf("%s: %d\n", i18n.T("order_label_extension_hours"), order.Content.ApprovalOrderExtensionHours)
		fmt.Printf("%s: %s\n", i18n.T("order_history_approved_duration"), approvedLockLabel(order.Content.ApprovedLock))
	}
	fmt.Printf("%s: %s\n", i18n.T("order_label_reviewer"), emptyDash(order.Reviewer.Username))
	fmt.Printf("%s: %s\n", i18n.T("order_label_review_notes"), emptyDash(order.ReviewNotes))
	if job != nil {
		fmt.Printf("\n%s: %s\n", i18n.T("table_job_name"), job.JobName)
		fmt.Printf("%s: %s\n", i18n.T("order_label_job_status"), job.Status)
		fmt.Printf("%s: %s\n", i18n.T("table_queue"), job.Queue)
		fmt.Printf("%s: %s\n", i18n.T("order_label_requested_resources"), formatResources(job.Resources))
	} else if jobErr != nil {
		fmt.Printf("\n%s\n", i18n.T("order_job_unavailable", jobErr.Message))
	}
}

func runOrderByName(cmd *cobra.Command, args []string) error {
	name, err := requiredArg(args, "order_label_name", "name")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	orders, err := client.ListApprovalOrdersByName(name, "")
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(
			os.Stdout,
			output.SuccessEnvelope(map[string]interface{}{"orders": orders}),
		)
	}
	printOrderTable(orders)
	return nil
}

func runOrderSubmit(cmd *cobra.Command, _ []string) error {
	req, err := collectOrderSubmitRequest(cmd)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	message, err := client.CreateApprovalOrder(req)
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeOrderMessage("order_submit_success", message)
}

func runOrderEdit(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	current, err := client.GetApprovalOrder(id, false)
	if err != nil {
		return cliErrFromAPI(err)
	}
	req, err := collectOrderEditRequest(cmd, current)
	if err != nil {
		return err
	}
	message, err := client.UpdateApprovalOrder(id, req)
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeOrderMessage("order_update_success", message)
}

func runOrderCancel(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	if err := requireConfirmation(cmd); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	message, err := client.DeleteApprovalOrder(id)
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeOrderMessage("order_delete_success", message)
}

func runAdminOrderApprove(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	lock, issues := collectOrderLockRequest(cmd)
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	notes, _ := cmd.Flags().GetString("review-notes")
	notes = strings.TrimSpace(notes)

	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	order, err := client.GetApprovalOrder(id, true)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if order.Type != "job" {
		if lock != nil {
			return errUsageFromIssues([]usageIssue{invalidIssue("lock", i18n.T("err_lock_requires_job"))})
		}
		return writeApprovalReview(client, id, "Approved", notes)
	}

	result, err := client.ApproveJobOrder(id, api.ApproveJobOrderRequest{Lock: lock, ReviewNotes: notes})
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		payload := map[string]interface{}{
			"message": result.Message, "lock_message": result.LockMessage, "approved_lock": result.ApprovedLock,
		}
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(payload))
	}
	fmt.Println(i18n.T("order_lock_success", result.LockMessage))
	fmt.Println(i18n.T("order_update_success", result.Message))
	return nil
}

func runAdminOrderReject(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	notes, _ := cmd.Flags().GetString("review-notes")
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return errUsageFromIssues([]usageIssue{missingIssue("review-notes", "flag_review-notes")})
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	return writeApprovalReview(client, id, "Rejected", notes)
}

func writeApprovalReview(client *api.Client, id uint, status, notes string) error {
	message, err := client.ReviewApprovalOrder(id, api.ApprovalOrderReviewRequest{Status: status, ReviewNotes: notes})
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeOrderMessage("order_update_success", message)
}

func runAdminOrderCheck(cmd *cobra.Command, _ []string) error {
	if err := requireConfirmation(cmd); err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	message, err := client.CheckApprovalOrders()
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeOrderMessage("order_check_success", message)
}

func collectOrderSubmitRequest(cmd *cobra.Command) (api.ApprovalOrderRequest, error) {
	name, _ := cmd.Flags().GetString("name")
	orderType, _ := cmd.Flags().GetString("type")
	typeID, _ := cmd.Flags().GetUint("type-id")
	reason, _ := cmd.Flags().GetString("reason")
	hours, _ := cmd.Flags().GetUint("hours")
	name = strings.TrimSpace(name)
	orderType = strings.TrimSpace(orderType)
	reason = strings.TrimSpace(reason)

	var issues []usageIssue
	if name == "" {
		issues = append(issues, missingIssue("name", "order_label_name"))
	}
	if orderType == "" {
		issues = append(issues, missingIssue("type", "order_label_type"))
	} else if !slices.Contains(orderTypes, orderType) {
		issues = append(issues, invalidIssue("type", i18n.T("err_invalid_order_type", orderType)))
	}
	if reason == "" {
		issues = append(issues, missingIssue("reason", "order_label_reason"))
	}
	if len(issues) > 0 {
		return api.ApprovalOrderRequest{}, errUsageFromIssues(issues)
	}
	return api.ApprovalOrderRequest{
		Name:           name,
		Type:           orderType,
		TypeID:         typeID,
		Reason:         reason,
		ExtensionHours: hours,
	}, nil
}

func collectOrderEditRequest(cmd *cobra.Command, current *api.ApprovalOrder) (api.ApprovalOrderRequest, error) {
	if current == nil {
		return api.ApprovalOrderRequest{}, errUsageFromIssues([]usageIssue{
			invalidIssue("id", i18n.T("err_order_not_loaded")),
		})
	}
	status, _ := cmd.Flags().GetString("status")
	name, _ := cmd.Flags().GetString("name")
	orderType, _ := cmd.Flags().GetString("type")
	typeID, _ := cmd.Flags().GetUint("type-id")
	reason, _ := cmd.Flags().GetString("reason")
	hours, _ := cmd.Flags().GetUint("hours")
	status = strings.TrimSpace(status)
	name = strings.TrimSpace(name)
	orderType = strings.TrimSpace(orderType)
	reason = strings.TrimSpace(reason)

	if status == "" {
		status = current.Status
	}
	if name == "" {
		name = current.Name
	}
	if orderType == "" {
		orderType = current.Type
	}
	if !cmd.Flags().Changed("type-id") {
		typeID = current.Content.ApprovalOrderTypeID
	}
	if reason == "" {
		reason = current.Content.ApprovalOrderReason
	}
	if !cmd.Flags().Changed("hours") {
		hours = current.Content.ApprovalOrderExtensionHours
	}

	var issues []usageIssue
	if !slices.Contains(orderEditableStatus, status) {
		issues = append(issues, invalidIssue("status", i18n.T("err_invalid_edit_status", status)))
	}
	if name == "" {
		issues = append(issues, missingIssue("name", "order_label_name"))
	}
	if !slices.Contains(orderTypes, orderType) {
		issues = append(issues, invalidIssue("type", i18n.T("err_invalid_order_type", orderType)))
	}
	if reason == "" {
		issues = append(issues, missingIssue("reason", "order_label_reason"))
	}
	if len(issues) > 0 {
		return api.ApprovalOrderRequest{}, errUsageFromIssues(issues)
	}

	return api.ApprovalOrderRequest{
		Name:           name,
		Type:           orderType,
		Status:         status,
		TypeID:         typeID,
		Reason:         reason,
		ExtensionHours: hours,
	}, nil
}

func collectOrderLockRequest(cmd *cobra.Command) (*api.JobOrderLockRequest, []usageIssue) {
	lockEnabled, _ := cmd.Flags().GetBool("lock")
	permanent, _ := cmd.Flags().GetBool("permanent")
	days, _ := cmd.Flags().GetInt("days")
	hours, _ := cmd.Flags().GetInt("hours")
	minutes, _ := cmd.Flags().GetInt("minutes")
	hasDuration := days != 0 || hours != 0 || minutes != 0
	if !lockEnabled {
		if permanent || hasDuration {
			return nil, []usageIssue{invalidIssue("lock", i18n.T("err_order_lock_flag_required"))}
		}
		return nil, nil
	}

	var issues []usageIssue
	if days < 0 || hours < 0 || minutes < 0 {
		issues = append(issues, invalidIssue("lock", i18n.T("err_lock_negative_duration")))
	}
	if permanent && hasDuration {
		issues = append(issues, invalidIssue("lock", i18n.T("err_order_lock_permanent_with_duration")))
	}
	if !permanent && !hasDuration {
		issues = append(issues, invalidIssue("lock", i18n.T("err_lock_duration")))
	}
	if len(issues) > 0 {
		return nil, issues
	}
	return &api.JobOrderLockRequest{IsPermanent: permanent, Days: days, Hours: hours, Minutes: minutes}, nil
}

func requireConfirmation(cmd *cobra.Command) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if viper.GetBool("no-interactive") && !yes {
		return &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrMissingRequiredFlag,
			Message:  i18n.T("err_confirm_required"),
		}
	}
	return nil
}

func writeOrderMessage(key string, message string) error {
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"message": message}))
	}
	fmt.Println(i18n.T(key, message))
	return nil
}

func printOrderTable(orders []api.ApprovalOrder) {
	fmt.Printf("%s %s %s %s %s %s\n",
		i18n.PadRight(i18n.T("table_id"), 8),
		i18n.PadRight(i18n.T("table_name"), 28),
		i18n.PadRight(i18n.T("table_type"), 16),
		i18n.PadRight(i18n.T("table_status"), 14),
		i18n.PadRight(i18n.T("table_owner"), 18),
		i18n.PadRight(i18n.T("table_created_at"), 22))
	for _, order := range orders {
		creator := order.Creator.Nickname
		if creator == "" {
			creator = order.Creator.Username
		}
		fmt.Printf("%s %s %s %s %s %s\n",
			i18n.PadRight(fmt.Sprint(order.ID), 8),
			i18n.PadRight(order.Name, 28),
			i18n.PadRight(order.Type, 16),
			i18n.PadRight(order.Status, 14),
			i18n.PadRight(creator, 18),
			i18n.PadRight(formatAPITime(order.CreatedAt), 22))
	}
}

func addOrderListFlags(cmd *cobra.Command, admin bool) {
	addListPaginationFlags(cmd)
	cmd.Flags().String("status", "", i18n.T("flag_status"))
	cmd.Flags().String("type", "", i18n.T("flag_type"))
	cmd.Flags().String("search", "", i18n.T("flag_search"))
	if admin {
		cmd.Flags().String("creator", "", i18n.T("flag_creator"))
	}
}

func init() {
	addOrderListFlags(orderLsCmd, false)

	orderSubmitCmd.Flags().String("name", "", "Approval target name")
	orderSubmitCmd.Flags().String("type", "job", "Approval order type")
	orderSubmitCmd.Flags().Uint("type-id", 0, "Approval target numeric ID")
	orderSubmitCmd.Flags().String("reason", "", "Approval reason")
	orderSubmitCmd.Flags().Uint("hours", 0, "Extension hours")

	orderEditCmd.Flags().String("name", "", "Approval target name")
	orderEditCmd.Flags().String("type", "", "Approval order type")
	orderEditCmd.Flags().Uint("type-id", 0, "Approval target numeric ID")
	orderEditCmd.Flags().String("reason", "", "Approval reason")
	orderEditCmd.Flags().Uint("hours", 0, "Extension hours")
	orderEditCmd.Flags().String("status", "Pending", "Approval order status")

	orderCancelCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")

	adminOrderApproveCmd.Flags().Bool("lock", false, "Lock the job before approving")
	adminOrderApproveCmd.Flags().Bool("permanent", false, "Lock the job permanently")
	adminOrderApproveCmd.Flags().Int("days", 0, "Lock duration days")
	adminOrderApproveCmd.Flags().Int("hours", 0, "Lock duration hours")
	adminOrderApproveCmd.Flags().Int("minutes", 0, "Lock duration minutes")
	adminOrderApproveCmd.Flags().String("review-notes", "", "Review notes")
	adminOrderRejectCmd.Flags().String("review-notes", "", "Review notes")
	adminOrderCheckCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	addOrderListFlags(adminOrderLsCmd, true)

	completion.RegisterFlagValue([]string{"order", "submit"}, "type", staticValueCompleter(orderTypes, nil))
	completion.RegisterFlagValue([]string{"order", "edit"}, "type", staticValueCompleter(orderTypes, nil))
	completion.RegisterFlagValue([]string{"order", "edit"}, "status", staticValueCompleter(orderEditableStatus, nil))
	completion.RegisterFlagValue([]string{"order", "ls"}, "status", staticValueCompleter(orderListStatuses, nil))
	completion.RegisterFlagValue([]string{"order", "ls"}, "type", staticValueCompleter(orderTypes, nil))
	completion.RegisterFlagValue([]string{"admin", "order", "ls"}, "status", staticValueCompleter(orderListStatuses, nil))
	completion.RegisterFlagValue([]string{"admin", "order", "ls"}, "type", staticValueCompleter(orderTypes, nil))

	orderCmd.AddCommand(orderLsCmd, orderGetCmd, orderByNameCmd, orderSubmitCmd, orderEditCmd, orderCancelCmd)
	rootCmd.AddCommand(orderCmd)
	adminOrderCmd.AddCommand(adminOrderLsCmd, adminOrderGetCmd, adminOrderApproveCmd, adminOrderRejectCmd, adminOrderCheckCmd)
	adminCmd.AddCommand(adminOrderCmd)
}
