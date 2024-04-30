package cliutils

import (
	"encoding/json"
	"fmt"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/commands/utils"
	artifactoryUtils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"os"
	"path"
)

type Result struct {
	SourcePath string `json:"sourcePath"`
	TargetPath string `json:"targetPath"`
	RtUrl      string `json:"rtUrl"`
}

type ResultsWrapper struct {
	Results []Result `json:"results"`
}

type GitHubActionSummary struct {
	dirPath     string                     // Directory path for the GitHubActionSummary data
	rawDataFile string                     // File which contains all the results of the commands
	uploadTree  *artifactoryUtils.FileTree // Upload a tree object to generate markdown
}

type Workflow struct {
	Name string `yaml:"name"`
	Jobs map[string]struct {
		Steps []map[string]interface{} `yaml:"steps"`
	} `yaml:"jobs"`
}

const (
	homeDir          = "/Users/runner/.jfrog/jfrog-github-summary"
	githubActionsEnv = "GITHUB_ACTIONS"
)

func GenerateGitHubActionSummary(result *utils.Result) (err error) {
	if os.Getenv(githubActionsEnv) != "true" {
		return
	}
	// Initiate the GitHubActionSummary, will check for previous runs and manage the runtime info.
	gh, err := initGithubActionSummary()
	if err != nil {
		return fmt.Errorf("failed while initiating Github job summaries: %w", err)
	}
	// Appends the current command results to the result file.
	if err = gh.AppendResult(result); err != nil {
		return fmt.Errorf("failed while appending results: %s", err)
	}
	// Generate upload tree
	if err = gh.generateUploadedFilesTree(); err != nil {
		return fmt.Errorf("failed while creating file tree: %w", err)
	}
	// Generate the whole markdown
	return gh.generateMarkdown()
}

func (gh *GitHubActionSummary) generateUploadedFilesTree() (err error) {
	object, _, err := gh.loadAndMarshalResultsFile()
	if err != nil {
		return
	}
	gh.uploadTree = artifactoryUtils.NewFileTree()
	for _, b := range object.Results {
		gh.uploadTree.AddFile(b.TargetPath)
	}
	return
}

func (gh *GitHubActionSummary) getRuntimeInfoFilePath() string {
	return path.Join(gh.dirPath, "runtime-info.json")
}

func (gh *GitHubActionSummary) getDataFilePath() string {
	return path.Join(gh.dirPath, gh.rawDataFile)
}

func (gh *GitHubActionSummary) AppendResult(result *utils.Result) error {
	// Read all the current command result files.
	var readContent []Result
	if result != nil && result.Reader() != nil {
		for _, file := range result.Reader().GetFilesPaths() {
			// Read source file
			sourceBytes, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			// Unmarshal source file content
			var sourceWrapper ResultsWrapper
			err = json.Unmarshal(sourceBytes, &sourceWrapper)
			if err != nil {
				return err
			}
			readContent = append(readContent, sourceWrapper.Results...)
		}
	}

	targetWrapper, targetBytes, err := gh.loadAndMarshalResultsFile()
	// Append source results to target results
	targetWrapper.Results = append(targetWrapper.Results, readContent...)

	// Marshal target results
	targetBytes, err = json.MarshalIndent(targetWrapper, "", "  ")
	if err != nil {
		return err
	}
	// Write target results to target file
	err = os.WriteFile(gh.getDataFilePath(), targetBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (gh *GitHubActionSummary) loadAndMarshalResultsFile() (targetWrapper ResultsWrapper, targetBytes []byte, err error) {
	// Load target file
	targetBytes, err = os.ReadFile(gh.getDataFilePath())
	if err != nil && !os.IsNotExist(err) {
		log.Warn("data file not found ", gh.getDataFilePath())
		return ResultsWrapper{}, nil, err
	}
	if len(targetBytes) <= 0 {
		log.Warn("empty data file: ", gh.getDataFilePath())
		return
	}
	// Unmarshal target file content, if it exists
	if err = json.Unmarshal(targetBytes, &targetWrapper); err != nil {
		return
	}
	return
}

func (gh *GitHubActionSummary) generateMarkdown() (err error) {

	tempMarkdownPath := path.Join(gh.dirPath, "github-action-summary.md")
	// Remove the file if it exists
	log.Info("removing previous file: ", tempMarkdownPath)
	_ = os.Remove(tempMarkdownPath)
	log.Info("writing markdown to: ", tempMarkdownPath)
	file, err := os.OpenFile(tempMarkdownPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		err = file.Close()
	}()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	// TODO handle errors better
	_, err = file.WriteString("# 🐸 JFrog CLI Github Action Summary 🐸\n")
	_, err = file.WriteString("## Uploaded artifacts:\n")
	_, err = file.WriteString("```\n" + gh.uploadTree.String() + "```")
	return
}

func (gh *GitHubActionSummary) createTempFile(filePath string, content any) (err error) {
	exists, err := fileutils.IsFileExists(filePath, true)
	if err != nil || exists {
		return
	}
	file, err := os.Create(filePath)
	defer func() {
		err = file.Close()
	}()
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	bytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}
	_, err = file.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}
	log.Info("created file:", file.Name())
	return
}

func initGithubActionSummary() (gh *GitHubActionSummary, err error) {
	gh, err = tryLoadPreviousRuntimeInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to load runtime info: %w", err)
	}
	if gh != nil {
		log.Debug("successfully loaded GitHubActionSummary from previous runs")
		return
	}
	log.Debug("creating new GitHubActionSummary...")
	gh, err = createNewGithubSummary()
	if err != nil {
		return nil, fmt.Errorf("failed to create temp files: %w", err)
	}
	return
}

// Loads previous steps information if exists
func tryLoadPreviousRuntimeInfo() (gh *GitHubActionSummary, err error) {
	gh = newGithubActionSummary(gh)
	if err = fileutils.CreateDirIfNotExist(homeDir); err != nil {
		return nil, fmt.Errorf("failed to create dir %s: %w", homeDir, err)
	}
	return
}

// Initializes a new GitHubActionSummary
func createNewGithubSummary() (gh *GitHubActionSummary, err error) {
	gh = newGithubActionSummary(gh)
	err = gh.createTempFile(gh.getDataFilePath(), ResultsWrapper{Results: []Result{}})
	if err != nil {
		return nil, fmt.Errorf("failed to create data file: %w", err)
	}
	return
}

func newGithubActionSummary(gh *GitHubActionSummary) *GitHubActionSummary {
	// TODO handle home dirs for each OS, and update the SetupCLI post action with this path.
	gh = &GitHubActionSummary{
		dirPath:     homeDir,
		rawDataFile: "data.json",
	}
	return gh
}
