//
// Copyright 2019 Chef Software, Inc.
// Author: Salim Afiune <afiune@chef.io>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chef/go-libs/config"
	"github.com/chef/go-libs/credentials"
	"github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/chef/chef-analyze/pkg/dist"
	"github.com/chef/chef-analyze/pkg/formatter"
	"github.com/chef/chef-analyze/pkg/reporting"
)

const (
	analyzeReportsDir = "reports" // Used for $HOME/.chef-workstation/reports
	analyzeErrorsDir  = "errors"  // Used for $HOME/.chef-workstation/errors
	repNameCookbooks  = "cookbooks"
	repNameNodes      = "nodes"
	ErrExt            = "err"
	TxtExt            = "txt"
	CsvExt            = "csv"
)

var (
	timestamp = time.Now().Format("20060102150405")
	reportCmd = &cobra.Command{
		Use:   "report",
		Short: fmt.Sprintf("Generate reports from a %s", dist.ServerProduct),
	}
	reportCookbooksCmd = &cobra.Command{
		Use:   "cookbooks",
		Short: "Generates a cookbook-oriented report",
		Args:  cobra.NoArgs,
		Long: `Generates a cookbook-oriented report containing details about the
upgrade compatibility errors and node cookbook usage.

The result is written to file.
`,
		RunE: func(_ *cobra.Command, _ []string) error {
			creds, err := credentials.FromViper(
				infraFlags.profile,
				overrideCredentials(),
			)

			if err != nil {
				return err
			}

			err = createOutputDirectories()
			if err != nil {
				return err
			}

			cfg := &reporting.Reporting{Credentials: creds}
			if infraFlags.noSSLverify {
				cfg.NoSSLVerify = true
			}

			chefClient, err := reporting.NewChefClient(cfg)
			if err != nil {
				return err
			}

			fmt.Printf("Finding available cookbooks...")
			cookbooksState, err := reporting.NewCookbooksReport(
				reporting.NewChefAnalyzeClient(chefClient),
				cookbooksFlags.runCookstyle,
				cookbooksFlags.onlyUnused,
				cookbooksFlags.workers,
				reportsFlags.nodeFilter,
				toAnonymize(),
			)

			if err != nil {
				return err
			}

			if cookbooksState.TotalCookbooks == 0 {
				fmt.Printf(" (0 found)\n\nNo cookbooks available for analysis.\n")
				return nil
			}
			fmt.Printf(" (%d found)\n", cookbooksState.TotalCookbooks)
			fmt.Println("Analyzing cookbooks...")

			progressBar := pb.New(cookbooksState.TotalCookbooks)
			progressBar.Start()
			go cookbooksState.Generate()
			for _ = range cookbooksState.Progress {
				progressBar.Increment()
			}
			progressBar.Finish()
			var (
				formattedSummary = formatter.CookbooksReportSummary(cookbooksState)
				results          *formatter.FormattedResult
				ext              string
			)

			fmt.Println(formattedSummary.Report)

			switch reportsFlags.format {
			case "csv":
				ext = CsvExt
				results = formatter.MakeCookbooksReportCSV(cookbooksState)
			default:
				ext = TxtExt
				results = formatter.MakeCookbooksReportTXT(cookbooksState)
			}

			err = saveReport(repNameCookbooks, ext, cookbooksState.NodeFilter, results.Report)
			if err != nil {
				return err
			}
			err = saveErrorReport(repNameCookbooks, results.Errors)
			if err != nil {
				return err
			}

			return nil
		},
	}
	reportNodesCmd = &cobra.Command{
		Use:   "nodes",
		Short: "Generates a nodes-oriented report",
		Long: `Generates a nodes-oriented report containing basic information about the node,
any applied policies, and the cookbooks used during the most recent chef-client run`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			creds, err := credentials.FromViper(
				infraFlags.profile,
				overrideCredentials(),
			)

			if err != nil {
				return err
			}

			err = createOutputDirectories()
			if err != nil {
				return err
			}

			cfg := &reporting.Reporting{Credentials: creds}
			if infraFlags.noSSLverify {
				cfg.NoSSLVerify = true
			}

			chefClient, err := reporting.NewChefClient(cfg)
			if err != nil {
				return err
			}

			fmt.Println("Analyzing nodes...")
			report, err := reporting.GenerateNodesReport(reporting.NewChefAnalyzeClient(chefClient), reportsFlags.nodeFilter, toAnonymize())
			if err != nil {
				return err
			}

			var (
				formattedSummary = formatter.NodesReportSummary(report, reportsFlags.nodeFilter)
				results          *formatter.FormattedResult
				ext              string
			)

			fmt.Println(formattedSummary.Report)

			switch reportsFlags.format {
			case "csv":
				ext = CsvExt
				results = formatter.MakeNodesReportCSV(report, reportsFlags.nodeFilter)
			default:
				ext = TxtExt
				results = formatter.MakeNodesReportTXT(report, reportsFlags.nodeFilter)
			}

			err = saveReport(repNameNodes, ext, reportsFlags.nodeFilter, results.Report)
			if err != nil {
				return err
			}
			err = saveErrorReport(repNameNodes, results.Errors)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cookbooksFlags struct {
		onlyUnused   bool
		runCookstyle bool
		workers      int
	}
	reportsFlags struct {
		format     string
		nodeFilter string
		anonymize  bool
	}
)

func init() {
	// Add shared infra/chef-server related flags
	addInfraFlagsToCommand(reportCmd)

	// common report flags
	reportCmd.PersistentFlags().StringVarP(
		&reportsFlags.format,
		"format", "f", "txt",
		"output format: txt is human readable, csv is machine readable",
	)
	reportCmd.PersistentFlags().StringVarP(
		&reportsFlags.nodeFilter,
		"node-filter", "F", "",
		"Search filter to apply to nodes",
	)

	// cookbooks cmd flags
	reportCookbooksCmd.PersistentFlags().IntVarP(
		&cookbooksFlags.workers,
		"workers", "w", 50,
		"maximum number of parallel workers at once",
	)
	reportCookbooksCmd.PersistentFlags().BoolVarP(
		&cookbooksFlags.onlyUnused,
		"only-unused", "u", false,
		"generate a report with only cookbooks that are not included in any node's runlist",
	)
	reportCookbooksCmd.PersistentFlags().BoolVarP(
		&cookbooksFlags.runCookstyle,
		"verify-upgrade", "V", false,
		"verify the upgrade compatibility of every cookbook",
	)
	reportCmd.PersistentFlags().BoolVarP(
		&reportsFlags.anonymize,
		"anonymize", "a", false,
		"replace cookbook and node names with hash values",
	)
	// adds the cookbooks command as a sub-command of the report command
	reportCmd.AddCommand(reportCookbooksCmd)

	// adds the nodes command as a sub-command of the report command
	reportCmd.AddCommand(reportNodesCmd)

	// adds the upload command as a hidden sub-command of the report command
	reportCmd.AddCommand(uploadCmd)

	// adds the session command as a hidden sub-command of the report command
	reportCmd.AddCommand(sessionCmd)
}

func createOutputDirectories() error {
	wsDir, err := config.ChefWorkstationDir()
	if err != nil {
		return err
	}

	var (
		reportsDir = filepath.Join(wsDir, analyzeReportsDir)
		errorsDir  = filepath.Join(wsDir, analyzeErrorsDir)
	)

	err = os.MkdirAll(reportsDir, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s/ directory", analyzeReportsDir)
	}
	err = os.MkdirAll(errorsDir, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s/ directory", analyzeErrorsDir)
	}
	return nil
}

func saveErrorReport(baseName string, content string) error {
	if content == "" {
		return nil
	}

	wsDir, err := config.ChefWorkstationDir()
	if err != nil {
		return err
	}

	var (
		errorsDir  = filepath.Join(wsDir, analyzeErrorsDir)
		reportName = fmt.Sprintf("%s-%s.%s", baseName, timestamp, "err")
		reportPath = filepath.Join(errorsDir, reportName)
	)
	reportFile, err := os.Create(reportPath)
	if err != nil {
		return errors.Wrap(err, "unable to save errors report")
	}

	reportFile.WriteString(content)
	reportFile.Close()

	fmt.Printf("Error report saved to %s\n", reportPath)
	return nil
}

func saveReport(baseName string, ext string, nodeFilter string, content string) error {
	if len(content) == 0 {
		return nil
	}

	wsDir, err := config.ChefWorkstationDir()
	if err != nil {
		return err
	}

	filtered := ""
	if len(nodeFilter) > 0 {
		filtered = "-filtered"
	}
	var (
		reportsDir = filepath.Join(wsDir, analyzeReportsDir)
		reportName = fmt.Sprintf("%s-%s%s.%s", baseName, timestamp, filtered, ext)
		reportPath = filepath.Join(reportsDir, reportName)
	)
	reportFile, err := os.Create(reportPath) // create a new report file
	if err != nil {
		return errors.Wrapf(err, "unable to save %s report", baseName)
	}

	reportFile.WriteString(content)
	reportFile.Close()

	fmt.Printf("%s report saved to %s\n", strings.Title(baseName), reportPath)
	return nil
}

func toAnonymize() bool {

	c, error := reporting.LoadConfig()
	if error != nil {
		return false
	}

	return reportsFlags.anonymize || c.Config.Reports.Anonymize

}
