package route53

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// Route53Client wraps AWS Route53 operations
type Route53Client struct {
	client       *route53.Client
	hostedZoneID string
	baseDomain   string
}

// NewRoute53Client creates a new Route53 client
func NewRoute53Client() (*Route53Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	hostedZoneID := os.Getenv("ROUTE53_HOSTED_ZONE_ID")
	if hostedZoneID == "" {
		return nil, fmt.Errorf("ROUTE53_HOSTED_ZONE_ID environment variable is not set")
	}

	baseDomain := os.Getenv("BASE_DOMAIN")
	if baseDomain == "" {
		baseDomain = "snapdeploy.app"
	}

	return &Route53Client{
		client:       route53.NewFromConfig(cfg),
		hostedZoneID: hostedZoneID,
		baseDomain:   baseDomain,
	}, nil
}

// DNSRecordRequest contains information for creating/updating DNS records
type DNSRecordRequest struct {
	Subdomain string // e.g., "my-app"
	Target    string // ALB DNS name or IP address
	Type      string // "A" or "CNAME"
}

// CreateOrUpdateRecord creates or updates a DNS record for a subdomain
func (c *Route53Client) CreateOrUpdateRecord(ctx context.Context, req DNSRecordRequest) error {
	// Construct full domain name
	fullDomain := fmt.Sprintf("%s.%s", req.Subdomain, c.baseDomain)

	// Determine record type - use ALIAS for ALB
	var change types.Change
	if strings.Contains(req.Target, ".elb.amazonaws.com") {
		// ALB target - use ALIAS record
		change = c.createAliasChange(fullDomain, req.Target)
	} else {
		// Regular CNAME or A record
		if req.Type == "A" {
			change = c.createARecordChange(fullDomain, req.Target)
		} else {
			change = c.createCNAMEChange(fullDomain, req.Target)
		}
	}

	// Create change batch
	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(c.hostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String(fmt.Sprintf("Upsert record for %s", fullDomain)),
			Changes: []types.Change{change},
		},
	}

	_, err := c.client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create/update DNS record: %w", err)
	}

	return nil
}

// createAliasChange creates an ALIAS record change for ALB
func (c *Route53Client) createAliasChange(fullDomain, albDNS string) types.Change {
	// Extract hosted zone ID from ALB DNS name
	// Format: my-alb-123456.us-east-1.elb.amazonaws.com
	region := extractRegionFromALB(albDNS)
	albHostedZoneID := getALBHostedZoneID(region)

	return types.Change{
		Action: types.ChangeActionUpsert,
		ResourceRecordSet: &types.ResourceRecordSet{
			Name: aws.String(fullDomain),
			Type: types.RRTypeA,
			AliasTarget: &types.AliasTarget{
				DNSName:              aws.String(albDNS),
				HostedZoneId:         aws.String(albHostedZoneID),
				EvaluateTargetHealth: true,
			},
		},
	}
}

// createCNAMEChange creates a CNAME record change
func (c *Route53Client) createCNAMEChange(fullDomain, target string) types.Change {
	return types.Change{
		Action: types.ChangeActionUpsert,
		ResourceRecordSet: &types.ResourceRecordSet{
			Name: aws.String(fullDomain),
			Type: types.RRTypeCname,
			TTL:  aws.Int64(300),
			ResourceRecords: []types.ResourceRecord{
				{Value: aws.String(target)},
			},
		},
	}
}

// createARecordChange creates an A record change
func (c *Route53Client) createARecordChange(fullDomain, ipAddress string) types.Change {
	return types.Change{
		Action: types.ChangeActionUpsert,
		ResourceRecordSet: &types.ResourceRecordSet{
			Name: aws.String(fullDomain),
			Type: types.RRTypeA,
			TTL:  aws.Int64(300),
			ResourceRecords: []types.ResourceRecord{
				{Value: aws.String(ipAddress)},
			},
		},
	}
}

// DeleteRecord deletes a DNS record
func (c *Route53Client) DeleteRecord(ctx context.Context, subdomain, recordType string) error {
	fullDomain := fmt.Sprintf("%s.%s", subdomain, c.baseDomain)

	// First, get the existing record
	listInput := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(c.hostedZoneID),
		StartRecordName: aws.String(fullDomain),
		StartRecordType: types.RRType(recordType),
		MaxItems:        aws.Int32(1),
	}

	listResult, err := c.client.ListResourceRecordSets(ctx, listInput)
	if err != nil {
		return fmt.Errorf("failed to list DNS records: %w", err)
	}

	if len(listResult.ResourceRecordSets) == 0 {
		return fmt.Errorf("record not found")
	}

	recordSet := &listResult.ResourceRecordSets[0]

	// Delete the record
	change := types.Change{
		Action:            types.ChangeActionDelete,
		ResourceRecordSet: recordSet,
	}

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(c.hostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String(fmt.Sprintf("Delete record for %s", fullDomain)),
			Changes: []types.Change{change},
		},
	}

	_, err = c.client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}

	return nil
}

// RecordExists checks if a DNS record exists
func (c *Route53Client) RecordExists(ctx context.Context, subdomain string) (bool, error) {
	fullDomain := fmt.Sprintf("%s.%s", subdomain, c.baseDomain)

	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(c.hostedZoneID),
		StartRecordName: aws.String(fullDomain),
		MaxItems:        aws.Int32(1),
	}

	result, err := c.client.ListResourceRecordSets(ctx, input)
	if err != nil {
		return false, fmt.Errorf("failed to check DNS record: %w", err)
	}

	if len(result.ResourceRecordSets) == 0 {
		return false, nil
	}

	// Check if the record name matches
	if *result.ResourceRecordSets[0].Name == fullDomain+"." {
		return true, nil
	}

	return false, nil
}

// extractRegionFromALB extracts AWS region from ALB DNS name
func extractRegionFromALB(albDNS string) string {
	// Format: my-alb-123456.us-east-1.elb.amazonaws.com
	parts := strings.Split(albDNS, ".")
	if len(parts) >= 3 {
		return parts[1]
	}
	// Default to us-east-1 if we can't parse
	return "us-east-1"
}

// getALBHostedZoneID returns the hosted zone ID for ALB in the given region
// These are AWS-managed hosted zone IDs for ALBs in each region
func getALBHostedZoneID(region string) string {
	// Reference: https://docs.aws.amazon.com/general/latest/gr/elb.html
	albHostedZones := map[string]string{
		"us-east-1":      "Z35SXDOTRQ7X7K",
		"us-east-2":      "Z3AADJGX6KTTL2",
		"us-west-1":      "Z368ELLRRE2KJ0",
		"us-west-2":      "Z1H1FL5HABSF5",
		"ca-central-1":   "ZQSVJUPU6J1EY",
		"eu-central-1":   "Z215JYRZR1TBD5",
		"eu-west-1":      "Z32O12XQLNTSW2",
		"eu-west-2":      "ZHURV8PSTC4K8",
		"eu-west-3":      "Z3Q77PNBQS71R4",
		"eu-north-1":     "Z23TAZ6LKFMNIO",
		"ap-northeast-1": "Z14GRHDCWA56QT",
		"ap-northeast-2": "ZWKZPGTI48KDX",
		"ap-northeast-3": "Z5LXEXXYW11ES",
		"ap-southeast-1": "Z1LMS91P8CMLE5",
		"ap-southeast-2": "Z1GM3OXH4ZPM65",
		"ap-south-1":     "ZP97RAFLXTNZK",
		"sa-east-1":      "Z2P70J7HTTTPLU",
	}

	if zoneID, ok := albHostedZones[region]; ok {
		return zoneID
	}

	// Default to us-east-1 if region not found
	return "Z35SXDOTRQ7X7K"
}

