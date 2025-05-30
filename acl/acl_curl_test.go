//go:build integration
// +build integration

/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package acl

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/stretchr/testify/require"

	"github.com/hypermodeinc/dgraph/v25/dgraphapi"
	"github.com/hypermodeinc/dgraph/v25/testutil"
	"github.com/hypermodeinc/dgraph/v25/x"
)

func (asuite *AclTestSuite) TestCurlAuthorization() {
	t := asuite.T()
	if testing.Short() {
		t.Skip("skipping because -short=true")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	gc, cleanup, err := asuite.dc.Client()
	require.NoError(t, err)
	defer cleanup()
	require.NoError(t, gc.LoginIntoNamespace(ctx, dgraphapi.DefaultUser,
		dgraphapi.DefaultPassword, x.RootNamespace))

	hc, err := asuite.dc.HTTPClient()
	require.NoError(t, err)
	require.NoError(t, hc.LoginIntoNamespace(dgraphapi.DefaultUser,
		dgraphapi.DefaultPassword, x.RootNamespace))
	createAccountAndData(t, gc, hc)

	// test query through curl
	require.NoError(t, hc.LoginIntoNamespace(userid, userpassword, x.RootNamespace))
	// No ACL rules are specified, so query should return empty response,
	// alter and mutate should fail.
	queryArgs := func(jwt string) []string {
		return []string{"-H", fmt.Sprintf("X-Dgraph-AccessToken:%s", jwt),
			"--ipv4",
			"-H", "Content-Type: application/dql",
			"-d", query, testutil.SockAddrHttp + "/query"}
	}
	testutil.VerifyCurlCmd(t, queryArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail: false,
	})

	mutateArgs := func(jwt string) []string {
		return []string{"-H", fmt.Sprintf("X-Dgraph-AccessToken:%s", jwt),
			"-H", "Content-Type: application/rdf",
			"-d", fmt.Sprintf(`{ set {
	   _:a <%s>  "string" .
	   }}`, predicateToWrite), testutil.SockAddrHttp + "/mutate"}

	}

	testutil.VerifyCurlCmd(t, mutateArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail:   true,
		DgraphErrMsg: "PermissionDenied",
	})

	alterArgs := func(jwt string) []string {
		return []string{"-H", fmt.Sprintf("X-Dgraph-AccessToken:%s", jwt),
			"-d", fmt.Sprintf(`%s: int .`, predicateToAlter), testutil.SockAddrHttp + "/alter"}
	}
	testutil.VerifyCurlCmd(t, alterArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail:   true,
		DgraphErrMsg: "PermissionDenied",
	})

	// sleep long enough (longer than 10s, the access JWT TTL defined in the docker-compose.yml
	// in this directory) for the accessJwt to expire, in order to test auto login through refresh
	// JWT
	glog.Infof("Sleeping for accessJwt to expire")
	time.Sleep(expireJwtSleep)
	testutil.VerifyCurlCmd(t, queryArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail:   true,
		DgraphErrMsg: "Token is expired",
	})
	testutil.VerifyCurlCmd(t, mutateArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail:   true,
		DgraphErrMsg: "Token is expired",
	})
	testutil.VerifyCurlCmd(t, alterArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail:   true,
		DgraphErrMsg: "Token is expired",
	})
	// login again using the refreshJwt
	require.NoError(t, hc.LoginUsingToken(x.RootNamespace))
	require.NoError(t, err, fmt.Sprintf("login through refresh httpToken failed: %v", err))
	hcWithGroot, err := asuite.dc.HTTPClient()
	require.NoError(t, err)
	require.NoError(t, hcWithGroot.LoginIntoNamespace(dgraphapi.DefaultUser,
		dgraphapi.DefaultPassword, x.RootNamespace))
	createGroupAndAcls(t, unusedGroup, false, hcWithGroot)
	time.Sleep(expireJwtSleep)
	testutil.VerifyCurlCmd(t, queryArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail:   true,
		DgraphErrMsg: "Token is expired",
	})
	// refresh the jwts again
	require.NoError(t, hc.LoginUsingToken(x.RootNamespace))

	require.NoError(t, err, fmt.Sprintf("login through refresh httpToken failed: %v", err))
	// verify that with an ACL rule defined, all the operations except query should
	// does not have the required permissions be denied when the acsess JWT
	testutil.VerifyCurlCmd(t, queryArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail: false,
	})
	testutil.VerifyCurlCmd(t, mutateArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail:   true,
		DgraphErrMsg: "PermissionDenied",
	})
	testutil.VerifyCurlCmd(t, alterArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail:   true,
		DgraphErrMsg: "PermissionDenied",
	})
	require.NoError(t, hcWithGroot.LoginIntoNamespace(dgraphapi.DefaultUser,
		dgraphapi.DefaultPassword, x.RootNamespace))
	createGroupAndAcls(t, devGroup, true, hcWithGroot)
	time.Sleep(defaultTimeToSleep)
	// refresh the jwts again
	require.NoError(t, hc.LoginUsingToken(x.RootNamespace))

	require.NoError(t, err, fmt.Sprintf("login through refresh httpToken failed: %v", err))
	// verify that the operations should be allowed again through the dev group
	testutil.VerifyCurlCmd(t, queryArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail: false,
	})
	testutil.VerifyCurlCmd(t, mutateArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail: false,
	})
	testutil.VerifyCurlCmd(t, alterArgs(hc.AccessJwt), &testutil.CurlFailureConfig{
		ShouldFail: false,
	})
}
