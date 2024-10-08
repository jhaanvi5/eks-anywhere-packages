package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/aws/eks-anywhere-packages/api/v1alpha1"
)

func TestNewBundleGenerate(t *testing.T) {
	tests := []struct {
		testname   string
		bundleName string
		wantBundle *api.PackageBundle
	}{
		{
			testname:   "TestNewBundleGenerate",
			bundleName: "bundlename",
			wantBundle: &api.PackageBundle{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PackageBundle",
					APIVersion: "packages.eks.amazonaws.com/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bundlename",
					Namespace: "eksa-packages",
					Annotations: map[string]string{
						"eksa.aws.com/excludes": "LnNwZWMucGFja2FnZXNbXS5zb3VyY2UucmVnaXN0cnkKLnNwZWMucGFja2FnZXNbXS5zb3VyY2UucmVwb3NpdG9yeQo=",
					},
				},
				Spec: api.PackageBundleSpec{
					Packages: []api.BundlePackage{
						{
							Name: "sample-package",
							Source: api.BundlePackageSource{
								Repository: "sample-Repository",
								Versions: []api.SourceVersion{
									{
										Name:   "v0.0",
										Digest: "sha256:da25f5fdff88c259bb2ce7c0f1e9edddaf102dc4fb9cf5159ad6b902b5194e66",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.testname, func(tt *testing.T) {
			got := NewBundleGenerate(tc.bundleName)
			if !reflect.DeepEqual(got, tc.wantBundle) {
				tt.Fatalf("GetClusterConfig() = %#v, want %#v", got, tc.wantBundle)
			}
		})
	}
}

var (
	testTagBundle      string = "0.1.0_latest_c4e25cb42e9bb88d2b8c2abfbde9f10ade39b214"
	testShaBundle      string = "sha256:d5467083c4d175e7e9bba823e95570d28fff86a2fbccb03f5ec3093db6f039be"
	testImageMediaType string = "application/vnd.oci.image.manifest.v1+json"
	testRegistryId     string = "067575901363.dkr.ecr.us-west-2.amazonaws.com"
	testRepositoryName string = "hello-eks-anywhere"
	testAccountID      string = "123456702424"
)

func TestNewPackageFromInput(t *testing.T) {
	client := newMockPrivateRegistryClientBundle(nil)
	stsclient := newMockSTSClient(nil)
	tests := []struct {
		client      *mockPrivateRegistryClientBundle
		testname    string
		testproject Project
		wantErr     bool
		wantBundle  *api.BundlePackage
	}{
		{
			testname: "Test no tags",
			testproject: Project{
				Name:       "hello-eks-anywhere",
				Repository: "hello-eks-anywhere",
				Registry:   testRegistryId,
				Versions:   []Tag{},
			},
			wantErr: true,
		},
		{
			testname: "Test named tag",
			testproject: Project{
				Name:       "hello-eks-anywhere",
				Repository: "hello-eks-anywhere",
				Registry:   testRegistryId,
				Versions: []Tag{
					{Name: testTagBundle},
				},
			},
			wantErr: false,
			wantBundle: &api.BundlePackage{
				Name: "hello-eks-anywhere",
				Source: api.BundlePackageSource{
					Repository: "hello-eks-anywhere",
					Registry:   testRegistryId,
					Versions: []api.SourceVersion{
						{
							Name:   testTagBundle,
							Digest: testShaBundle,
						},
					},
				},
			},
		},
		{
			testname: "Test 'latest' tag",
			testproject: Project{
				Name:       "hello-eks-anywhere",
				Repository: "hello-eks-anywhere",
				Registry:   testRegistryId,
				Versions: []Tag{
					{Name: "latest"},
				},
			},
			wantErr: false,
			wantBundle: &api.BundlePackage{
				Name: "hello-eks-anywhere",
				Source: api.BundlePackageSource{
					Repository: "hello-eks-anywhere",
					Registry:   testRegistryId,
					Versions: []api.SourceVersion{
						{
							Name:   testTagBundle,
							Digest: testShaBundle,
						},
					},
				},
			},
		},
		{
			testname: "Test '-latest' in the middle of tag",
			testproject: Project{
				Name:       "hello-eks-anywhere",
				Repository: "hello-eks-anywhere",
				Registry:   testRegistryId,
				Versions: []Tag{
					{Name: "test-latest-helm"},
				},
			},
			wantErr: false,
			wantBundle: &api.BundlePackage{
				Name: "hello-eks-anywhere",
				Source: api.BundlePackageSource{
					Repository: "hello-eks-anywhere",
					Registry:   testRegistryId,
					Versions: []api.SourceVersion{
						{
							Name:   "test-latest-helm",
							Digest: testShaBundle,
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.testname, func(tt *testing.T) {
			clients := &SDKClients{
				ecrClient: &ecrClient{
					registryClient: client,
				},
				stsClient: &stsClient{
					stsClientInterface: stsclient,
				},
			}
			got, err := clients.NewPackageFromInput(tc.testproject)
			if (err != nil) != tc.wantErr {
				tt.Fatalf("NewPackageFromInput() error = %v, wantErr %v got %v", err, tc.wantErr, got)
			}
			if !reflect.DeepEqual(got, tc.wantBundle) {
				tt.Fatalf("NewPackageFromInput() = %#v\n\n\n, want %#v", got, tc.wantBundle)
			}
		})
	}
}

type mockPrivateRegistryClientBundle struct {
	err error
}

func newMockPrivateRegistryClientBundle(err error) *mockPrivateRegistryClientBundle {
	return &mockPrivateRegistryClientBundle{
		err: err,
	}
}

func (r *mockPrivateRegistryClientBundle) DescribeImages(ctx context.Context, params *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeImagesOutput, error) {
	if r.err != nil {
		return nil, r.err
	}
	testImagePushedAt := time.Now()
	return &ecr.DescribeImagesOutput{
		ImageDetails: []ecrtypes.ImageDetail{
			{
				ImageDigest:            &testShaBundle,
				ImageTags:              []string{testTagBundle},
				ImageManifestMediaType: &testImageMediaType,
				ImagePushedAt:          &testImagePushedAt,
				RegistryId:             &testRegistryId,
				RepositoryName:         &testRepositoryName,
			},
		},
	}, nil
}

func (r *mockPrivateRegistryClientBundle) DescribeRegistry(ctx context.Context, params *ecr.DescribeRegistryInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRegistryOutput, error) {
	panic("not implemented") // TODO: Implement
}

func (r *mockPrivateRegistryClientBundle) GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error) {
	panic("not implemented") // TODO: Implement
}

type mockSTSClient struct {
	err error
}

func newMockSTSClient(err error) *mockSTSClient {
	return &mockSTSClient{
		err: err,
	}
}

func (r *mockSTSClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &sts.GetCallerIdentityOutput{
		Account: &testAccountID,
	}, nil
}
