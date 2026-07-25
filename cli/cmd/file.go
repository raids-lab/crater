package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

var fileRemoteRoots = []string{"user", "public", "account"}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "View remote files",
	Long:  "List files in user, public, and account storage spaces.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var fileLsCmd = &cobra.Command{
	Use:   "ls [remote-path]",
	Short: "List remote files",
	Args:  maxOneArg,
	RunE:  runFileLs,
}

func runFileLs(_ *cobra.Command, args []string) error {
	remotePath := ""
	if len(args) == 1 {
		remotePath = args[0]
	}
	normalizedPath, err := normalizeRemotePath(remotePath, true)
	if err != nil {
		return err
	}

	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	files, err := client.ListFiles(normalizedPath)
	if err != nil {
		return cliErrFromAPI(err)
	}
	sortFileInfos(files)

	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"files": files,
		}))
	}
	printFileTable(files)
	return nil
}

func normalizeRemotePath(remotePath string, allowEmpty bool) (string, error) {
	if remotePath == "" {
		if allowEmpty {
			return "", nil
		}
		return "", invalidRemotePathIssue(i18n.T("err_file_path_invalid", remotePath))
	}
	if strings.ContainsRune(remotePath, '\\') {
		return "", invalidRemotePathIssue(i18n.T("err_file_path_invalid", remotePath))
	}
	for _, character := range remotePath {
		if unicode.IsControl(character) {
			return "", invalidRemotePathIssue(i18n.T("err_file_path_invalid", remotePath))
		}
	}

	normalized := strings.Trim(remotePath, "/")
	if normalized == "" {
		if allowEmpty {
			return "", nil
		}
		return "", invalidRemotePathIssue(i18n.T("err_file_path_empty"))
	}
	rawSegments := strings.Split(normalized, "/")
	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		if segment == ".." {
			return "", invalidRemotePathIssue(i18n.T("err_file_path_invalid", remotePath))
		}
		if segment == "" || segment == "." {
			continue
		}
		segments = append(segments, segment)
	}
	if len(segments) == 0 {
		if allowEmpty {
			return "", nil
		}
		return "", invalidRemotePathIssue(i18n.T("err_file_path_invalid", remotePath))
	}
	if !isFileRemoteRoot(segments[0]) {
		return "", invalidRemotePathIssue(i18n.T("err_file_path_root", remotePath))
	}
	return strings.Join(segments, "/"), nil
}

func invalidRemotePathIssue(message string) error {
	return errUsageFromIssues([]usageIssue{{
		Code:    errorcodes.ErrInvalidFlagValue,
		Message: message,
		Field:   "remote-path",
	}})
}

func isFileRemoteRoot(value string) bool {
	for _, root := range fileRemoteRoots {
		if value == root {
			return true
		}
	}
	return false
}

func sortFileInfos(files []api.FileInfo) {
	sort.SliceStable(files, func(left, right int) bool {
		if files[left].IsDir != files[right].IsDir {
			return files[left].IsDir
		}
		leftName := strings.ToLower(files[left].Name)
		rightName := strings.ToLower(files[right].Name)
		if leftName == rightName {
			return files[left].Name < files[right].Name
		}
		return leftName < rightName
	})
}

func printFileTable(files []api.FileInfo) {
	fmt.Printf("%s %s %s %s\n",
		i18n.PadRight(i18n.T("table_name"), 36),
		i18n.PadRight(i18n.T("table_type"), 12),
		i18n.PadRight(i18n.T("file_table_size"), 14),
		i18n.PadRight(i18n.T("file_table_modified"), 22))
	for _, file := range files {
		fileType := i18n.T("file_type_regular")
		size := strconv.FormatInt(file.Size, 10)
		if file.IsDir {
			fileType = i18n.T("file_type_directory")
			size = "-"
		}
		modified := "-"
		if !file.ModifyTime.IsZero() {
			modified = file.ModifyTime.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%s %s %s %s\n",
			i18n.PadRight(displayFileName(file.Name), 36),
			i18n.PadRight(fileType, 12),
			i18n.PadRight(size, 14),
			i18n.PadRight(modified, 22))
	}
}

func displayFileName(name string) string {
	for _, character := range name {
		if unicode.IsControl(character) {
			return strconv.QuoteToGraphic(name)
		}
	}
	return name
}

func fileRootCompleter(ctx completion.Context) ([]completion.Candidate, error) {
	prefix := strings.ToLower(completion.CurrentWordPrefix(ctx))
	candidates := make([]completion.Candidate, 0, len(fileRemoteRoots))
	for _, root := range fileRemoteRoots {
		value := root
		if prefix != "" && !strings.HasPrefix(value, prefix) {
			continue
		}
		candidates = append(candidates, completion.Candidate{
			Value:       value,
			Description: i18n.T("file_root_" + root + "_desc"),
		})
	}
	return candidates, nil
}

func init() {
	fileCmd.AddCommand(fileLsCmd)
	rootCmd.AddCommand(fileCmd)
	completion.RegisterPositional([]string{"file", "ls"}, 0, fileRootCompleter)
}
