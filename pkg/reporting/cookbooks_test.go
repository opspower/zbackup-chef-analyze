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

package reporting_test

import (
	"os"
	"testing"

	chef "github.com/chef/go-chef"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	subject "github.com/chef/chef-analyze/pkg/reporting"
)

func TestCookbooksRecordCorrectableAndOffenses(t *testing.T) {
	csr := subject.CookbookRecord{
		Files: []subject.CookbookFile{
			subject.CookbookFile{
				Offenses: []subject.CookstyleOffense{
					subject.CookstyleOffense{Correctable: false},
				},
			},
			subject.CookbookFile{
				Offenses: []subject.CookstyleOffense{
					subject.CookstyleOffense{Correctable: true},
					subject.CookstyleOffense{Correctable: false},
					subject.CookstyleOffense{Correctable: true},
					subject.CookstyleOffense{Correctable: true},
				},
			},
		},
	}
	assert.Equal(t, 3, csr.NumCorrectable())
	assert.Equal(t, 5, csr.NumOffenses())
}

func TestCookbooksEmpty(t *testing.T) {
	c, err := subject.NewCookbooks(
		newMockCookbook(chef.CookbookListResult{}, nil, nil),
		makeMockSearch("[]", nil),
		false,
	)
	assert.Nil(t, err)
	if assert.NotNil(t, c) {
		assert.Empty(t, c.Records)
		assert.Equal(t, 0, c.TotalCookbooks)
	}
}

// Given a valid set of cookbooks in use by nodes,
// And the --only-unused flag is provided,
// Verify that the result set is as expected.
func TestCookbooksInUseDisplayOnlyUnused(t *testing.T) {
	cookbookList := chef.CookbookListResult{
		"foo": chef.CookbookVersions{
			Versions: []chef.CookbookVersion{
				chef.CookbookVersion{Version: "0.1.0"},
				chef.CookbookVersion{Version: "0.2.0"},
				chef.CookbookVersion{Version: "0.3.0"},
			},
		},
		"bar": chef.CookbookVersions{
			Versions: []chef.CookbookVersion{
				chef.CookbookVersion{Version: "0.1.0"},
			},
		},
	}
	c, err := subject.NewCookbooks(
		newMockCookbook(cookbookList, nil, nil),
		makeMockSearch(mockedNodesSearchRows(), nil), // nodes are found
		true, // display only unused cookbooks
	)
	assert.Nil(t, err)
	if assert.NotNil(t, c) {
		assert.Empty(t, c.Records)
		assert.Equal(t, 4, c.TotalCookbooks)
	}

	c, err = subject.NewCookbooks(
		newMockCookbook(cookbookList, nil, nil),
		makeMockSearch(mockedNodesSearchRows(), nil), // nodes are found
		false, // display only used cookbooks
	)
	assert.Nil(t, err)
	if assert.NotNil(t, c) {
		assert.Equal(t, 4, c.TotalCookbooks)
		if assert.Equal(t, 4, len(c.Records)) {
			// we check for only one bar and 3 foo cookbooks
			var (
				foo = 0
				bar = 0
			)

			for _, rec := range c.Records {
				switch rec.Name {
				case "foo":
					foo++
				case "bar":
					bar++
				default:
					t.Fatal("unexpected cookbook name")
				}

				assert.NotEmpty(t, rec.Nodes, "nodes should be found")
			}

			assert.Equal(t, 3, foo, "unexpected number of cookbooks foo")
			assert.Equal(t, 1, bar, "unexpected number of cookbooks bar")
		}
	}
}

// Given a valid set of cookbooks that are not being used by any node,
// And the --only-unused flag is provided,
// Verify that the result set is as expected.
func TestCookbooksNotUsedDisplayOnlyUnused(t *testing.T) {
	cookbookList := chef.CookbookListResult{
		"foo": chef.CookbookVersions{
			Versions: []chef.CookbookVersion{
				chef.CookbookVersion{Version: "0.1.0"},
				chef.CookbookVersion{Version: "0.2.0"},
				chef.CookbookVersion{Version: "0.3.0"},
			},
		},
		"bar": chef.CookbookVersions{
			Versions: []chef.CookbookVersion{
				chef.CookbookVersion{Version: "0.1.0"},
			},
		},
	}
	c, err := subject.NewCookbooks(
		newMockCookbook(cookbookList, nil, nil),
		makeMockSearch("[]", nil), // no nodes are returned
		true,                      // display only unused cookbooks
	)
	assert.Nil(t, err)
	if assert.NotNil(t, c) {
		assert.Equal(t, 4, c.TotalCookbooks)
		if assert.Equal(t, 4, len(c.Records)) {
			// we check for only one bar and 3 foo cookbooks
			var (
				foo = 0
				bar = 0
			)

			for _, rec := range c.Records {
				switch rec.Name {
				case "foo":
					foo++
				case "bar":
					bar++
				default:
					t.Fatal("unexpected cookbook name")
				}

				assert.Empty(t, rec.Nodes, "no nodes should be found")
			}

			assert.Equal(t, 3, foo, "unexpected number of cookbooks foo")
			assert.Equal(t, 1, bar, "unexpected number of cookbooks bar")
		}
	}

	c, err = subject.NewCookbooks(
		newMockCookbook(cookbookList, nil, nil),
		makeMockSearch("[]", nil), // no nodes are returned
		false,                     // display only used cookbooks
	)
	assert.Nil(t, err)
	if assert.NotNil(t, c) {
		assert.Empty(t, c.Records)
		assert.Equal(t, 4, c.TotalCookbooks)
	}

}

// Given a valid set of cookbooks in use by nodes,
// verify that the result set is as expected.
func TestCookbooks(t *testing.T) {
	savedPath := setupBinstubsDir()
	defer os.Setenv("PATH", savedPath)

	cookbookList := chef.CookbookListResult{
		"foo": chef.CookbookVersions{
			Versions: []chef.CookbookVersion{
				chef.CookbookVersion{Version: "0.1.0"},
				chef.CookbookVersion{Version: "0.2.0"},
				chef.CookbookVersion{Version: "0.3.0"},
			},
		},
		"bar": chef.CookbookVersions{
			Versions: []chef.CookbookVersion{
				chef.CookbookVersion{Version: "0.1.0"},
			},
		},
	}
	c, err := subject.NewCookbooks(
		// TODO @afiune gotta implement the newMockCookbook desiredCookbookList
		newMockCookbook(cookbookList, nil, nil),
		makeMockSearch(mockedNodesSearchRows(), nil),
		false,
	)
	assert.Nil(t, err)
	if assert.NotNil(t, c) {
		assert.Equal(t, 4, c.TotalCookbooks)
		if assert.Equal(t, 4, len(c.Records)) {
			// we check for only one bar and 3 foo cookbooks
			var (
				foo = 0
				bar = 0
			)

			for _, rec := range c.Records {
				switch rec.Name {
				case "foo":
					foo++
				case "bar":
					bar++
				default:
					t.Fatal("unexpected cookbook name")
				}
			}

			assert.Equal(t, 3, foo, "unexpected number of cookbooks foo")
			assert.Equal(t, 1, bar, "unexpected number of cookbooks bar")
		}
	}
}

// Given a failure trying to get the list of cookbooks,
// verify the result set is as expected
func TestCookbooks_ListCookbooksErrors(t *testing.T) {
	c, err := subject.NewCookbooks(
		newMockCookbook(chef.CookbookListResult{}, errors.New("i/o timeout"), nil),
		makeMockSearch("[]", nil),
		false,
	)
	if assert.NotNil(t, err) {
		assert.Equal(t, "unable to retrieve cookbooks: i/o timeout", err.Error())
	}
	assert.Nil(t, c)
}

// Given a failure in downloading a cookbook, verify
// the result set is as expected
func TestCookbooks_DownloadErrors(t *testing.T) {
}

// Given a failure in fetching node usage for a cookbook,
// the result set is as expected
func TestCookbooks_UsageStatErrors(t *testing.T) {
}
