package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

type rawReadSpec struct {
	PayloadKey string
	Path       string
	Params     func(*cobra.Command) map[string]string
	Table      func(interface{})
}

func runRawRead(cmd *cobra.Command, spec rawReadSpec) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	data, err := client.GetRaw(spec.Path, spec.Params(cmd))
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON || spec.Table == nil {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			spec.PayloadKey: data,
		}))
	}
	spec.Table(data)
	return nil
}

func runRawStringRead(cmd *cobra.Command, path string, params map[string]string, payloadKey string) error {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	data, err := client.GetRawString(path, params)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			payloadKey: data,
		}))
	}
	fmt.Print(data)
	if data != "" && !strings.HasSuffix(data, "\n") {
		fmt.Println()
	}
	_ = cmd
	return nil
}

func noParams(*cobra.Command) map[string]string {
	return nil
}

func getBoolParam(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetBool(name)
	if v {
		return "true"
	}
	return "false"
}

func getStringParam(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return strings.TrimSpace(v)
}

func getIntParam(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetInt(name)
	return strconv.Itoa(v)
}

func requiredUintArg(args []string, labelKey string, field string) (uint, error) {
	value, err := requiredArg(args, labelKey, field)
	if err != nil {
		return 0, err
	}
	parsed, parseErr := strconv.ParseUint(value, 10, 0)
	if parseErr != nil {
		return 0, errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_uint_arg", i18n.T(labelKey), value),
			Field:   field,
		}})
	}
	return uint(parsed), nil
}

func rawList(data interface{}) []map[string]interface{} {
	items, ok := data.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if ok {
			out = append(out, m)
		}
	}
	return out
}

func rawMap(data interface{}) map[string]interface{} {
	m, _ := data.(map[string]interface{})
	return m
}

func rawString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func rawNestedString(m map[string]interface{}, key, nested string) string {
	child, _ := m[key].(map[string]interface{})
	return rawString(child, nested)
}

func rawResourceSummary(m map[string]interface{}, key string) string {
	child, _ := m[key].(map[string]interface{})
	if child == nil {
		return "-"
	}
	parts := []string{}
	for _, field := range []string{"used", "running", "pending", "limit"} {
		if v := rawString(child, field); v != "" {
			parts = append(parts, field+"="+v)
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}

func printSimpleTable(data interface{}, cols ...string) {
	rows := rawList(data)
	if len(cols) == 0 {
		return
	}
	for _, col := range cols {
		fmt.Print(i18n.PadRight(strings.ToUpper(col), 18))
	}
	fmt.Println()
	for _, row := range rows {
		for _, col := range cols {
			fmt.Print(i18n.PadRight(emptyDash(rawString(row, col)), 18))
		}
		fmt.Println()
	}
}
