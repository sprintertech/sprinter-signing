// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package topology

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"

	"github.com/sprintertech/sprinter-signing/config/relayer"
	"github.com/sprintertech/sprinter-signing/topology"
)

var (
	testTopologyCMD = &cobra.Command{
		Use:   "test",
		Short: "Test topology from S3",
		Long: "CLI tests does provided S3 bucket contain topology that could be well " +
			"decrypted with provided password and then parsed accordingly",
		RunE: testTopology,
	}
)

var (
	url           string
	region        string
	endpoint      string
	accessKey     string
	secretKey     string
	hash          string
	decryptionKey string
	staging       bool
)

func init() {
	testTopologyCMD.PersistentFlags().StringVar(&decryptionKey, "decryption-key", "", "password to decrypt topology")
	_ = testTopologyCMD.MarkFlagRequired("decryption-key")
	testTopologyCMD.PersistentFlags().StringVar(&url, "url", "", "S3 bucket name")
	_ = testTopologyCMD.MarkFlagRequired("url")
	testTopologyCMD.PersistentFlags().StringVar(&region, "region", "nyc3", "S3 region")
	testTopologyCMD.PersistentFlags().StringVar(&endpoint, "endpoint", "https://fra1.digitaloceanspaces.com", "S3 endpoint")
	testTopologyCMD.PersistentFlags().StringVar(&accessKey, "access-key", "", "S3 access key")
	_ = testTopologyCMD.MarkFlagRequired("access-key")
	testTopologyCMD.PersistentFlags().StringVar(&secretKey, "secret-key", "", "S3 secret key")
	_ = testTopologyCMD.MarkFlagRequired("secret-key")
	testTopologyCMD.PersistentFlags().StringVar(&hash, "hash", "", "hash of topology")
	testTopologyCMD.PersistentFlags().BoolVar(&staging, "staging", false, "use staging topology path")
}

func testTopology(cmd *cobra.Command, args []string) error {
	config := relayer.TopologyConfiguration{
		EncryptionKey: decryptionKey,
		Url:           url,
		Region:        region,
		Endpoint:      endpoint,
		Path:          "",
	}

	doCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(config.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)
	if err != nil {
		return err
	}
	s3Client := s3.NewFromConfig(doCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(config.Endpoint)
	})

	nt, err := topology.NewNetworkTopologyProvider(config, s3Client, staging)
	if err != nil {
		return err
	}
	decryptedTopology, err := nt.NetworkTopology(hash)
	if err != nil {
		return err
	}

	fmt.Printf("Everything is fine your topology is \n")
	fmt.Printf("%+v", decryptedTopology)
	return nil
}
