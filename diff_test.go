// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges_test

import (
	jujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/bundlechanges"
)

type diffSuite struct {
	jujutesting.IsolationSuite
}

var _ = gc.Suite(&diffSuite{})

func (s *diffSuite) TestDiffing1(c *gc.C) {
	c.Fatalf("writeme")
}

func (s *diffSuite) TestNewDiffEmpty(c *gc.C) {
	diff := bundlechanges.NewBundleDiff()
	c.Assert(diff.Empty(), jc.IsTrue)
}

func (s *diffSuite) TestApplicationsNotEmpty(c *gc.C) {
	diff := bundlechanges.NewBundleDiff()
	diff.Applications["mantell"] = &bundlechanges.ApplicationDiff{
		Missing: bundlechanges.ModelSide,
	}
	c.Assert(diff.Empty(), jc.IsFalse)
}

func (s *diffSuite) TestMachinesNotEmpty(c *gc.C) {
	diff := bundlechanges.NewBundleDiff()
	diff.Machines["1"] = &bundlechanges.MachineDiff{
		Missing: bundlechanges.BundleSide,
	}
	c.Assert(diff.Empty(), jc.IsFalse)
}

func (s *diffSuite) TestSeriesNotEmpty(c *gc.C) {
	diff := bundlechanges.NewBundleDiff()
	diff.Series = &bundlechanges.StringDiff{"xenial", "bionic"}
	c.Assert(diff.Empty(), jc.IsFalse)
}

func (s *diffSuite) TestRelationsNotEmpty(c *gc.C) {
	diff := bundlechanges.NewBundleDiff()
	diff.Relations = &bundlechanges.RelationDiff{
		BundleExtra: [][]string{
			{"sinkane:telephone", "bad-sav:hensteeth"},
		},
	}
	c.Assert(diff.Empty(), jc.IsFalse)
}
