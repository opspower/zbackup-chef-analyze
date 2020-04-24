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

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelpCommand(t *testing.T) {
	out, err, exitcode := ChefAnalyze("help")
	assert.Contains(t,
		out.String(),
		"Use \"chef-analyze [command] --help\" for more information about a command.",
		"STDOUT bottom message doesn't match")
	assert.Empty(t,
		err.String(),
		"STDERR should be empty")
	assert.Equal(t, 0, exitcode,
		"EXITCODE is not the expected one")
}

func TestHelpFlags_h(t *testing.T) {
	out, err, exitcode := ChefAnalyze("-h")
	assert.Contains(t,
		out.String(),
		"Use \"chef-analyze [command] --help\" for more information about a command.",
		"STDOUT bottom message doesn't match")
	assert.Empty(t,
		err.String(),
		"STDERR should be empty")
	assert.Equal(t, 0, exitcode,
		"EXITCODE is not the expected one")
}

func TestHelpFlags__help(t *testing.T) {
	out, err, exitcode := ChefAnalyze("--help")
	assert.Contains(t,
		out.String(),
		"Use \"chef-analyze [command] --help\" for more information about a command.",
		"STDOUT bottom message doesn't match")
	assert.Empty(t,
		err.String(),
		"STDERR should be empty")
	assert.Equal(t, 0, exitcode,
		"EXITCODE is not the expected one")
}

func TestHelpNoArgs(t *testing.T) {
	out, err, exitcode := ChefAnalyze()
	assert.Contains(t,
		out.String(),
		"Use \"chef-analyze [command] --help\" for more information about a command.",
		"STDOUT bottom message doesn't match")
	assert.Empty(t,
		err.String(),
		"STDERR should be empty")
	assert.Equal(t, 0, exitcode,
		"EXITCODE is not the expected one")
}

func TestHelpCommandDisplayHelpFromCommand(t *testing.T) {
	out, err, exitcode := ChefAnalyze("help", "report")
	assert.Contains(t, out.String(),
		"--chef-server-url",
		"STDOUT chef-server-url flag doesn't exist")
	assert.Contains(t, out.String(),
		"--client-key",
		"STDOUT client-key flag doesn't exist")
	assert.Contains(t, out.String(),
		"--client-name",
		"STDOUT client-name flag doesn't exist")
	assert.Contains(t, out.String(),
		"--credentials",
		"STDOUT credentials flag doesn't exist")
	assert.Contains(t, out.String(),
		"--help",
		"STDOUT help flag doesn't exist")
	assert.Contains(t, out.String(),
		"--profile",
		"STDOUT profile flag doesn't exist")
	assert.Contains(t,
		out.String(),
		"chef-analyze report [command]",
		"STDOUT missing help about the report sub-command")
	assert.Empty(t,
		err.String(),
		"STDERR should be empty")
	assert.Equal(t, 0, exitcode,
		"EXITCODE is not the expected one")
}

func TestHelpCommandDisplayHelpFromUnknownCommand(t *testing.T) {
	out, err, exitcode := ChefAnalyze("help", "foo")
	// NOTE since this is an unknown command, we should display the help
	// message via STDERR and not STDOUT
	assert.Contains(t,
		err.String(),
		"Use \"chef-analyze [command] --help\" for more information about a command.",
		"STDERR bottom message doesn't match")
	assert.Empty(t,
		out.String(),
		"STDOUT should be empty")
	assert.Equal(t, 0, exitcode,
		"EXITCODE is not the expected one")
}
