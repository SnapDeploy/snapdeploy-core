package alb

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// ALBClient wraps AWS Application Load Balancer operations
type ALBClient struct {
	client      *elasticloadbalancingv2.Client
	listenerArn string
	vpcID       string
}

// NewALBClient creates a new ALB client
func NewALBClient() (*ALBClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	listenerArn := os.Getenv("ALB_LISTENER_ARN")
	if listenerArn == "" {
		return nil, fmt.Errorf("ALB_LISTENER_ARN environment variable is not set")
	}

	vpcID := os.Getenv("VPC_ID")
	if vpcID == "" {
		return nil, fmt.Errorf("VPC_ID environment variable is not set")
	}

	return &ALBClient{
		client:      elasticloadbalancingv2.NewFromConfig(cfg),
		listenerArn: listenerArn,
		vpcID:       vpcID,
	}, nil
}

// CreateTargetGroupAndRule creates a target group and listener rule for a deployment
func (c *ALBClient) CreateTargetGroupAndRule(ctx context.Context, serviceName, customDomain, baseDomain string, containerPort int32) (string, error) {
	// Create target group
	targetGroupArn, err := c.createTargetGroup(ctx, serviceName, containerPort)
	if err != nil {
		return "", fmt.Errorf("failed to create target group: %w", err)
	}

	// Create listener rule for the subdomain
	fullDomain := fmt.Sprintf("%s.%s", customDomain, baseDomain)
	if err := c.createListenerRule(ctx, fullDomain, targetGroupArn, serviceName); err != nil {
		// If rule creation fails, try to clean up target group
		c.deleteTargetGroup(ctx, targetGroupArn)
		return "", fmt.Errorf("failed to create listener rule: %w", err)
	}

	return targetGroupArn, nil
}

// createTargetGroup creates or updates a target group for a service
func (c *ALBClient) createTargetGroup(ctx context.Context, serviceName string, port int32) (string, error) {
	// Check if target group already exists
	existingGroups, err := c.findTargetGroupsByName(ctx, serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to check existing target groups: %w", err)
	}

	// If target group exists, check if port matches
	if len(existingGroups) > 0 {
		existingTG := existingGroups[0]
		existingPort := aws.ToInt32(existingTG.Port)

		if existingPort == port {
			// Port matches, reuse existing target group
			log.Printf("[ALB] Reusing existing target group %s with port %d", serviceName, port)
			return *existingTG.TargetGroupArn, nil
		}

		// Port doesn't match, need to recreate with new port
		// IMPORTANT: Must delete listener rules FIRST, then target group
		log.Printf("[ALB] Target group %s exists with port %d, but need port %d. Recreating...", serviceName, existingPort, port)

		// Step 1: Delete all listener rules using this target group
		log.Printf("[ALB] Deleting listener rules for %s...", serviceName)
		rules, err := c.findRulesByServiceName(ctx, serviceName)
		if err != nil {
			return "", fmt.Errorf("failed to find listener rules: %w", err)
		}

		for _, rule := range rules {
			isDefault := rule.IsDefault != nil && *rule.IsDefault
			if rule.RuleArn != nil && !isDefault {
				if err := c.deleteListenerRule(ctx, *rule.RuleArn); err != nil {
					log.Printf("[ALB] Warning: failed to delete listener rule: %v", err)
				} else {
					log.Printf("[ALB] Deleted listener rule: %s", *rule.RuleArn)
				}
			}
		}

		// Step 2: Now delete the target group
		log.Printf("[ALB] Deleting old target group %s...", serviceName)
		if err := c.deleteTargetGroup(ctx, *existingTG.TargetGroupArn); err != nil {
			return "", fmt.Errorf("failed to delete old target group: %w", err)
		}
		log.Printf("[ALB] Successfully deleted old target group")
	}

	// Create new target group
	input := &elasticloadbalancingv2.CreateTargetGroupInput{
		Name:                       aws.String(serviceName),
		Protocol:                   types.ProtocolEnumHttp,
		Port:                       aws.Int32(port),
		VpcId:                      aws.String(c.vpcID),
		TargetType:                 types.TargetTypeEnumIp,
		HealthCheckEnabled:         aws.Bool(true),
		HealthCheckPath:            aws.String("/"),
		HealthCheckProtocol:        types.ProtocolEnumHttp,
		HealthCheckIntervalSeconds: aws.Int32(30),
		HealthCheckTimeoutSeconds:  aws.Int32(5),
		HealthyThresholdCount:      aws.Int32(2),
		UnhealthyThresholdCount:    aws.Int32(2),
		Matcher: &types.Matcher{
			HttpCode: aws.String("200-399"),
		},
	}

	result, err := c.client.CreateTargetGroup(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create target group: %w", err)
	}

	if len(result.TargetGroups) == 0 {
		return "", fmt.Errorf("no target group created")
	}

	log.Printf("[ALB] Created new target group %s with port %d", serviceName, port)
	return *result.TargetGroups[0].TargetGroupArn, nil
}

// createListenerRule creates or updates an ALB listener rule for host-based routing
func (c *ALBClient) createListenerRule(ctx context.Context, hostHeader, targetGroupArn, serviceName string) error {
	// Check if a rule already exists for this service
	existingRules, err := c.findRulesByServiceName(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to check existing rules: %w", err)
	}

	// If rule exists, update it to point to the new target group
	if len(existingRules) > 0 {
		for _, rule := range existingRules {
			if rule.RuleArn != nil {
				// Update existing rule to point to new target group
				modifyInput := &elasticloadbalancingv2.ModifyRuleInput{
					RuleArn: rule.RuleArn,
					Actions: []types.Action{
						{
							Type:           types.ActionTypeEnumForward,
							TargetGroupArn: aws.String(targetGroupArn),
						},
					},
				}

				_, err := c.client.ModifyRule(ctx, modifyInput)
				if err != nil {
					return fmt.Errorf("failed to update listener rule: %w", err)
				}

				log.Printf("[ALB] Updated existing listener rule for %s", serviceName)
				return nil
			}
		}
	}

	// Find the next available priority
	priority, err := c.findNextPriority(ctx)
	if err != nil {
		return fmt.Errorf("failed to find available priority: %w", err)
	}

	// Create new rule
	input := &elasticloadbalancingv2.CreateRuleInput{
		ListenerArn: aws.String(c.listenerArn),
		Priority:    aws.Int32(priority),
		Conditions: []types.RuleCondition{
			{
				Field: aws.String("host-header"),
				HostHeaderConfig: &types.HostHeaderConditionConfig{
					Values: []string{hostHeader},
				},
			},
		},
		Actions: []types.Action{
			{
				Type:           types.ActionTypeEnumForward,
				TargetGroupArn: aws.String(targetGroupArn),
			},
		},
		Tags: []types.Tag{
			{
				Key:   aws.String("ServiceName"),
				Value: aws.String(serviceName),
			},
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("SnapDeploy"),
			},
		},
	}

	_, err = c.client.CreateRule(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create listener rule: %w", err)
	}

	log.Printf("[ALB] Created new listener rule for %s", serviceName)
	return nil
}

// findNextPriority finds the next available priority for a listener rule
func (c *ALBClient) findNextPriority(ctx context.Context) (int32, error) {
	input := &elasticloadbalancingv2.DescribeRulesInput{
		ListenerArn: aws.String(c.listenerArn),
	}

	result, err := c.client.DescribeRules(ctx, input)
	if err != nil {
		return 0, err
	}

	// Find the highest priority (priorities 1-50000, lower number = higher priority)
	// We'll start from 1000 and go up to leave room for manual rules
	maxPriority := int32(999)
	for _, rule := range result.Rules {
		if rule.Priority != nil {
			if priority, err := strconv.Atoi(*rule.Priority); err == nil {
				if int32(priority) > maxPriority && int32(priority) < 50000 {
					maxPriority = int32(priority)
				}
			}
		}
	}

	return maxPriority + 1, nil
}

// DeleteTargetGroupAndRule deletes the target group and listener rule for a service
func (c *ALBClient) DeleteTargetGroupAndRule(ctx context.Context, serviceName string) error {
	// Find listener rule by tags
	rules, err := c.findRulesByServiceName(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to find listener rules: %w", err)
	}

	// Delete listener rules
	for _, rule := range rules {
		// Skip default rule
		isDefault := rule.IsDefault != nil && *rule.IsDefault
		if rule.RuleArn != nil && !isDefault {
			err := c.deleteListenerRule(ctx, *rule.RuleArn)
			if err != nil {
				return fmt.Errorf("failed to delete listener rule: %w", err)
			}
		}
	}

	// Find and delete target group
	targetGroups, err := c.findTargetGroupsByName(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to find target groups: %w", err)
	}

	for _, tg := range targetGroups {
		if tg.TargetGroupArn != nil {
			err := c.deleteTargetGroup(ctx, *tg.TargetGroupArn)
			if err != nil {
				return fmt.Errorf("failed to delete target group: %w", err)
			}
		}
	}

	return nil
}

// findRulesByServiceName finds listener rules by service name tag
func (c *ALBClient) findRulesByServiceName(ctx context.Context, serviceName string) ([]types.Rule, error) {
	input := &elasticloadbalancingv2.DescribeRulesInput{
		ListenerArn: aws.String(c.listenerArn),
	}

	result, err := c.client.DescribeRules(ctx, input)
	if err != nil {
		return nil, err
	}

	var matchingRules []types.Rule
	for _, rule := range result.Rules {
		if rule.RuleArn != nil {
			// Get tags for this rule
			tagsInput := &elasticloadbalancingv2.DescribeTagsInput{
				ResourceArns: []string{*rule.RuleArn},
			}
			tagsResult, err := c.client.DescribeTags(ctx, tagsInput)
			if err != nil {
				continue
			}

			// Check if ServiceName tag matches
			for _, tagDesc := range tagsResult.TagDescriptions {
				for _, tag := range tagDesc.Tags {
					if tag.Key != nil && *tag.Key == "ServiceName" &&
						tag.Value != nil && *tag.Value == serviceName {
						matchingRules = append(matchingRules, rule)
						break
					}
				}
			}
		}
	}

	return matchingRules, nil
}

// findTargetGroupsByName finds target groups by name
func (c *ALBClient) findTargetGroupsByName(ctx context.Context, name string) ([]types.TargetGroup, error) {
	input := &elasticloadbalancingv2.DescribeTargetGroupsInput{
		Names: []string{name},
	}

	result, err := c.client.DescribeTargetGroups(ctx, input)
	if err != nil {
		// Target group might not exist
		return nil, nil
	}

	return result.TargetGroups, nil
}

// deleteListenerRule deletes a listener rule
func (c *ALBClient) deleteListenerRule(ctx context.Context, ruleArn string) error {
	input := &elasticloadbalancingv2.DeleteRuleInput{
		RuleArn: aws.String(ruleArn),
	}

	_, err := c.client.DeleteRule(ctx, input)
	return err
}

// deleteTargetGroup deletes a target group
func (c *ALBClient) deleteTargetGroup(ctx context.Context, targetGroupArn string) error {
	input := &elasticloadbalancingv2.DeleteTargetGroupInput{
		TargetGroupArn: aws.String(targetGroupArn),
	}

	_, err := c.client.DeleteTargetGroup(ctx, input)
	return err
}
