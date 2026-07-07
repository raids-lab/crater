package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	orderTypes          = []string{"job", "dataset"}
	orderEditableStatus = []string{"Pending"}
	orderReviewStatuses = []string{"Approved", "Rejected", "Canceled"}
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

var orderLsCmd = &cobra.Command{Use: "ls", Short: "List approval orders", Args: noArgs, RunE: runOrderLs}
var orderGetCmd = &cobra.Command{Use: "get <id>", Short: "Get an approval order", Args: exactArgs(1, "id"), RunE: runOrderGet}
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
	return runRawRead(cmd, rawReadSpec{PayloadKey: "orders", Path: api.ApprovalOrderPrefix, Params: noParams, Table: printOrderTable})
}

func runAdminOrderLs(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "orders", Path: api.AdminApprovalPrefix, Params: noParams, Table: printOrderTable})
}

func runOrderGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "order", Path: fmt.Sprintf("%s/%s", api.ApprovalOrderPrefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runAdminOrderGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "order", Path: fmt.Sprintf("%s/%s", api.AdminApprovalPrefix, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runOrderByName(cmd *cobra.Command, args []string) error {
	name, err := requiredArg(args, "order_label_name", "name")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "orders", Path: api.ApprovalOrderPrefix + "/name/" + name, Params: noParams, Table: printOrderTable})
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
	message, lockMessage, lockEnabled, err := reviewApprovalOrder(cmd, id, "Approved")
	if err != nil {
		return err
	}
	if outputJSON {
		payload := map[string]interface{}{"message": message}
		if lockEnabled {
			payload["lock_message"] = lockMessage
		}
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(payload))
	}
	if lockEnabled {
		fmt.Println(i18n.T("order_lock_success", lockMessage))
	}
	fmt.Println(i18n.T("order_update_success", message))
	return nil
}

func runAdminOrderReject(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "order_label_id", "id")
	if err != nil {
		return err
	}
	message, _, _, err := reviewApprovalOrder(cmd, id, "Rejected")
	if err != nil {
		return err
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

func reviewApprovalOrder(cmd *cobra.Command, id uint, status string) (message string, lockMessage string, lockEnabled bool, err error) {
	notes, _ := cmd.Flags().GetString("review-notes")
	notes = strings.TrimSpace(notes)

	var issues []usageIssue
	if !slices.Contains(orderReviewStatuses, status) {
		issues = append(issues, invalidIssue("status", i18n.T("err_invalid_status", status)))
	}
	if status == "Rejected" && notes == "" {
		issues = append(issues, missingIssue("review-notes", "flag_review-notes"))
	}
	lockReq, lockEnabled, lockIssues := collectOrderLockRequest(cmd, nil)
	issues = append(issues, lockIssues...)
	if len(issues) > 0 {
		return "", "", false, errUsageFromIssues(issues)
	}

	client, clientErr := activeAPIClient()
	if clientErr != nil {
		return "", "", false, clientErr
	}

	order, apiErr := client.GetApprovalOrder(id, true)
	if apiErr != nil {
		return "", "", false, cliErrFromAPI(apiErr)
	}

	lockReq.Name = order.Name
	typeIssues := validateOrderLockTarget(lockEnabled, order)
	issues = append(issues, typeIssues...)
	if len(issues) > 0 {
		return "", "", false, errUsageFromIssues(issues)
	}

	if lockEnabled {
		lockMessage, err = client.LockJob(lockReq)
		if err != nil {
			return "", "", false, cliErrFromAPI(err)
		}
	}
	message, err = client.ReviewApprovalOrder(id, api.ApprovalOrderReviewRequest{
		Status:      status,
		ReviewNotes: notes,
	})
	if err != nil {
		return "", "", false, cliErrFromAPI(err)
	}
	return message, lockMessage, lockEnabled, nil
}

func collectOrderLockRequest(cmd *cobra.Command, order *api.ApprovalOrder) (api.JobLockRequest, bool, []usageIssue) {
	lockEnabled, _ := cmd.Flags().GetBool("lock")
	permanent, _ := cmd.Flags().GetBool("permanent")
	days, _ := cmd.Flags().GetInt("days")
	lockHours, _ := cmd.Flags().GetInt("hours")
	minutes, _ := cmd.Flags().GetInt("minutes")
	if !lockEnabled {
		return api.JobLockRequest{}, false, nil
	}

	var issues []usageIssue
	if days < 0 || lockHours < 0 || minutes < 0 {
		issues = append(issues, invalidIssue("lock", i18n.T("err_lock_negative_duration")))
	}
	if !permanent && days == 0 && lockHours == 0 && minutes == 0 {
		issues = append(issues, invalidIssue("lock", i18n.T("err_lock_duration")))
	}
	if len(issues) > 0 {
		return api.JobLockRequest{}, false, issues
	}

	return api.JobLockRequest{
		IsPermanent: permanent,
		Days:        days,
		Hours:       lockHours,
		Minutes:     minutes,
	}, true, nil
}

func validateOrderLockTarget(lockEnabled bool, order *api.ApprovalOrder) []usageIssue {
	if !lockEnabled {
		return nil
	}
	if order == nil {
		return []usageIssue{invalidIssue("id", i18n.T("err_order_not_loaded"))}
	}
	if order.Type != "job" {
		return []usageIssue{invalidIssue("lock", i18n.T("err_lock_requires_job"))}
	}
	return nil
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

func missingIssue(field string, labelKey string) usageIssue {
	return usageIssue{
		Code:    errorcodes.ErrMissingRequiredFlag,
		Message: i18n.T("err_missing_required", i18n.T(labelKey), field),
		Field:   field,
	}
}

func invalidIssue(field string, message string) usageIssue {
	return usageIssue{Code: errorcodes.ErrInvalidFlagValue, Message: message, Field: field}
}

func printOrderTable(data interface{}) {
	fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(i18n.T("table_id"), 8), i18n.PadRight(i18n.T("table_name"), 28), i18n.PadRight(i18n.T("table_type"), 16), i18n.PadRight(i18n.T("table_status"), 14), i18n.PadRight("CREATOR", 18), i18n.PadRight("CREATED", 22))
	for _, row := range rawList(data) {
		fmt.Printf("%s %s %s %s %s %s\n", i18n.PadRight(rawString(row, "id"), 8), i18n.PadRight(rawString(row, "name"), 28), i18n.PadRight(rawString(row, "type"), 16), i18n.PadRight(rawString(row, "status"), 14), i18n.PadRight(rawNestedString(row, "creator", "nickname"), 18), i18n.PadRight(rawString(row, "createdAt"), 22))
	}
}

func init() {
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

	completion.RegisterFlagValue([]string{"order", "submit"}, "type", staticValueCompleter(orderTypes, nil))
	completion.RegisterFlagValue([]string{"order", "edit"}, "type", staticValueCompleter(orderTypes, nil))
	completion.RegisterFlagValue([]string{"order", "edit"}, "status", staticValueCompleter(orderEditableStatus, nil))

	orderCmd.AddCommand(orderLsCmd, orderGetCmd, orderByNameCmd, orderSubmitCmd, orderEditCmd, orderCancelCmd)
	rootCmd.AddCommand(orderCmd)
	adminOrderCmd.AddCommand(adminOrderLsCmd, adminOrderGetCmd, adminOrderApproveCmd, adminOrderRejectCmd, adminOrderCheckCmd)
	adminCmd.AddCommand(adminOrderCmd)
}
