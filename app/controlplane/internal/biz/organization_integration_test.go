//
// Copyright 2023 The Chainloop Authors.
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

package biz_test

import (
	"context"
	"testing"

	"github.com/chainloop-dev/chainloop/app/controlplane/internal/biz"
	"github.com/chainloop-dev/chainloop/app/controlplane/internal/biz/testhelpers"
	"github.com/chainloop-dev/chainloop/app/controlplane/plugins/sdk/v1"
	integrationMocks "github.com/chainloop-dev/chainloop/app/controlplane/plugins/sdk/v1/mocks"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/chainloop-dev/chainloop/internal/credentials"
	creds "github.com/chainloop-dev/chainloop/internal/credentials/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// and delete cascades that we want to validate that they work too
func (s *OrgIntegrationTestSuite) TestCreate() {
	ctx := context.Background()

	testCases := []struct {
		name          string
		expectedError bool
	}{
		// autogenerated name
		{"", false},
		{"a", false},
		{"aa-aa", false},
		{"-aaa", true},
		// no under-scores
		{"aaa_aaa", true},
		{"1-aaaa", false},
		{"Aaaaa", true},
		{"12-foo-bar-waz", false},
		// 63 max
		{"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijk", false},
		// over the max size
		{"aabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijk", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			org, err := s.Organization.Create(ctx, tc.name)
			if tc.expectedError {
				s.Error(err)
				return
			}

			require.NoError(s.T(), err)
			if tc.name == "" {
				// It was autogenerated
				s.NotEmpty(org.Name)
			} else {
				s.Equal(tc.name, org.Name)
			}
		})
	}
}

func (s *OrgIntegrationTestSuite) TestUpdate() {
	ctx := context.Background()

	s.T().Run("invalid org ID", func(t *testing.T) {
		// Invalid org ID
		_, err := s.Organization.Update(ctx, s.user.ID, "invalid", nil)
		s.Error(err)
		s.True(biz.IsErrInvalidUUID(err))
	})

	s.T().Run("org non existent", func(t *testing.T) {
		// org not found
		_, err := s.Organization.Update(ctx, s.user.ID, uuid.NewString(), nil)
		s.Error(err)
		s.True(biz.IsNotFound(err))
	})

	s.T().Run("org not accessible to user", func(t *testing.T) {
		org2, err := s.Organization.Create(ctx, "testing-org")
		require.NoError(s.T(), err)
		_, err = s.Organization.Update(ctx, s.user.ID, org2.ID, nil)
		s.Error(err)
		s.True(biz.IsNotFound(err))
	})

	s.T().Run("valid name update", func(t *testing.T) {
		got, err := s.Organization.Update(ctx, s.user.ID, s.org.ID, toPtrS("new-name"))
		s.NoError(err)
		s.Equal("new-name", got.Name)
	})

	s.T().Run("invalid name update", func(t *testing.T) {
		_, err := s.Organization.Update(ctx, s.user.ID, s.org.ID, toPtrS("invalid_new-name"))
		s.Error(err)
		s.True(biz.IsErrValidation(err))
	})
}

// We are doing an integration test here because there are some database constraints
// and delete cascades that we want to validate that they work too
func (s *OrgIntegrationTestSuite) TestDeleteOrg() {
	assert := assert.New(s.T())
	ctx := context.Background()

	s.T().Run("invalid org ID", func(t *testing.T) {
		// Invalid org ID
		err := s.Organization.Delete(ctx, "invalid")
		assert.Error(err)
		assert.True(biz.IsErrInvalidUUID(err))
	})

	s.T().Run("org non existent", func(t *testing.T) {
		// org not found
		err := s.Organization.Delete(ctx, uuid.NewString())
		assert.Error(err)
		assert.True(biz.IsNotFound(err))
	})

	s.T().Run("org, integrations and repositories deletion", func(t *testing.T) {
		// Mock calls to credentials deletion for both the integration and the OCI repository
		s.mockedCredsReaderWriter.On("DeleteCredentials", ctx, "stored-OCI-secret").Return(nil)

		err := s.Organization.Delete(ctx, s.org.ID)
		assert.NoError(err)

		// Integrations and repo deleted as well
		integrations, err := s.Integration.List(ctx, s.org.ID)
		assert.NoError(err)
		assert.Empty(integrations)

		ociRepo, err := s.CASBackend.FindDefaultBackend(ctx, s.org.ID)
		assert.Nil(ociRepo)
		assert.ErrorAs(err, &biz.ErrNotFound{})

		workflows, err := s.Workflow.List(ctx, s.org.ID)
		assert.NoError(err)
		assert.Empty(workflows)

		contracts, err := s.WorkflowContract.List(ctx, s.org.ID)
		assert.NoError(err)
		assert.Empty(contracts)
	})
}

// Run the tests
func TestOrgUseCase(t *testing.T) {
	suite.Run(t, new(OrgIntegrationTestSuite))
}

// Utility struct to hold the test suite
type OrgIntegrationTestSuite struct {
	testhelpers.UseCasesEachTestSuite
	org                     *biz.Organization
	user                    *biz.User
	mockedCredsReaderWriter *creds.ReaderWriter
}

func (s *OrgIntegrationTestSuite) SetupTest() {
	t := s.T()
	var err error
	assert := assert.New(s.T())
	ctx := context.Background()

	// Override credentials writer to set expectations
	s.mockedCredsReaderWriter = creds.NewReaderWriter(t)
	// Mock API call to store credentials

	// OCI repository credentials
	s.mockedCredsReaderWriter.On(
		"SaveCredentials", ctx, mock.Anything, &credentials.OCIKeypair{Repo: "repo", Username: "username", Password: "pass"},
	).Return("stored-OCI-secret", nil)

	s.TestingUseCases = testhelpers.NewTestingUseCases(t, testhelpers.WithCredsReaderWriter(s.mockedCredsReaderWriter))

	// Create org, integration and oci repository
	s.org, err = s.Organization.Create(ctx, "testing-org")
	assert.NoError(err)

	s.user, err = s.User.FindOrCreateByEmail(ctx, "foo@test.com")
	assert.NoError(err)
	_, err = s.Membership.Create(ctx, s.org.ID, s.user.ID, true)
	assert.NoError(err)

	// Integration
	// Mocked integration that will return both generic configuration and credentials
	integration := integrationMocks.NewFanOut(s.T())
	integration.On("Describe").Return(&sdk.IntegrationInfo{})
	integration.On("ValidateRegistrationRequest", mock.Anything).Return(nil)
	integration.On("Register", ctx, mock.Anything).Return(&sdk.RegistrationResponse{
		Configuration: []byte("deadbeef")}, nil)

	config, err := structpb.NewStruct(map[string]interface{}{"firstName": "John"})
	assert.NoError(err)

	_, err = s.Integration.RegisterAndSave(ctx, s.org.ID, "", integration, config)
	assert.NoError(err)

	// OCI repository
	_, err = s.CASBackend.CreateOrUpdate(ctx, s.org.ID, "repo", "username", "pass", backendType, true)
	assert.NoError(err)

	// Workflow + contract
	_, err = s.Workflow.Create(ctx, &biz.WorkflowCreateOpts{Name: "test workflow", OrgID: s.org.ID})
	assert.NoError(err)

	// check integration, OCI repository and workflow and contracts are present in the db
	integrations, err := s.Integration.List(ctx, s.org.ID)
	assert.NoError(err)
	assert.Len(integrations, 1)

	ociRepo, err := s.CASBackend.FindDefaultBackend(ctx, s.org.ID)
	assert.NoError(err)
	assert.NotNil(ociRepo)

	workflows, err := s.Workflow.List(ctx, s.org.ID)
	assert.NoError(err)
	assert.Len(workflows, 1)

	contracts, err := s.WorkflowContract.List(ctx, s.org.ID)
	assert.NoError(err)
	assert.Len(contracts, 1)
}
