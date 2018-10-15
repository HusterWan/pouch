package main

import (
	"strings"

	"github.com/alibaba/pouch/test/command"
	"github.com/alibaba/pouch/test/environment"

	"github.com/go-check/check"
	"github.com/gotestyourself/gotestyourself/icmd"
)

// PouchPluginSuite is the test suite for ps CLI.
type PouchPluginSuite struct{}

func init() {
	check.Suite(&PouchPluginSuite{})
}

// SetUpSuite does common setup in the beginning of each test suite.
func (suite *PouchPluginSuite) SetUpSuite(c *check.C) {
	SkipIfFalse(c, environment.IsLinux)

	environment.PruneAllContainers(apiClient)
	PullImage(c, busyboxImage)
}

// TearDownTest does cleanup work in the end of each test.
func (suite *PouchPluginSuite) TearDownTest(c *check.C) {
}

func (suite *PouchRunSuite) TestRunQuotaId(c *check.C) {
	if !environment.IsDiskQuota() {
		c.Skip("Host does not support disk quota")
	}
	name := "TestRunQuotaId"
	Id := "16777216"

	res := command.PouchRun("run", "-d", "--name", name, "--label", "DiskQuota=10G", "--label", "QuotaId="+Id, busyboxImage, "top")
	defer DelContainerForceMultyTime(c, name)
	res.Assert(c, icmd.Success)

	output := command.PouchRun("inspect", "-f", "{{.Config.Labels.QuotaId}}", name).Stdout()
	c.Assert(strings.TrimSpace(output), check.Equals, Id)

}

func (suite *PouchRunSuite) TestRunAutoQuotaId(c *check.C) {
	if !environment.IsDiskQuota() {
		c.Skip("Host does not support disk quota")
	}
	name := "TestRunAutoQuotaId"
	AutoQuotaIdValue := "true"

	res := command.PouchRun("run", "-d", "--name", name, "--label", "DiskQuota=10G", "--label", "AutoQuotaId="+AutoQuotaIdValue, busyboxImage, "top")
	defer DelContainerForceMultyTime(c, name)
	res.Assert(c, icmd.Success)

	output := command.PouchRun("inspect", "-f", "{{.Config.Labels.AutoQuotaId}}", name).Stdout()
	c.Assert(strings.TrimSpace(output), check.Equals, AutoQuotaIdValue)
}
