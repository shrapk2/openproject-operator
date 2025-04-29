package controller

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
)

// BuildInventoryMarkdownReport generates a markdown report from CloudInventoryReport
func BuildInventoryMarkdownReport(rep *v1alpha1.CloudInventoryReport) string {
	var b strings.Builder
	b.WriteString("## Cloud Inventory Report\n")

	// Container images section
	if len(rep.Status.ContainerImages) > 0 {
		cluster := rep.Status.ContainerImages[0].Cluster
		b.WriteString(fmt.Sprintf("- Cluster: `%s`\n", cluster))
		b.WriteString(fmt.Sprintf("- Total Pods: `%d`\n", rep.Status.Summary["pods"]))
		b.WriteString(fmt.Sprintf("- Unique Images: `%d`\n\n", rep.Status.Summary["images"]))

		b.WriteString("#### Image Details:\n")
		for _, img := range rep.Status.ContainerImages {
			b.WriteString(fmt.Sprintf("- `%s`\n", img.Image))
			b.WriteString(fmt.Sprintf("  - Repo: `%s`\n", img.Repository))
			b.WriteString(fmt.Sprintf("  - Tag: `%s`\n", img.Version))
			if img.SHA != "" {
				b.WriteString(fmt.Sprintf("  - SHA256: `%s`\n", img.SHA))
			}
		}

		b.WriteString("\n#### CSV Summary\n```csv\n")
		b.WriteString("cluster,image,repository,version,sha\n")
		for _, img := range rep.Status.ContainerImages {
			b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
				img.Cluster, img.Image, img.Repository, img.Version, img.SHA))
		}
		b.WriteString("```\n")
	}

	// Process all services dynamically
	for _, service := range monitoredServices {
		count := service.GetCount(rep)
		if count > 0 {
			b.WriteString(fmt.Sprintf("\n### AWS %s Inventory\n", service.Name))

			// Call the appropriate formatter based on service name
			switch service.Name {
			case "EC2":
				formatEC2Summary(&b, rep)
			case "RDS":
				formatRDSSummary(&b, rep)
			case "ELBV2":
				formatELBV2Summary(&b, rep)
			case "S3":
				formatS3Summary(&b, rep)
			case "EIP": // ← new
				formatEIPSummary(&b, rep)
			case "ECR":
				formatECRSummary(&b, rep)
			case "NATGateways":
				formatNATGatewaysSummary(&b, rep)
			case "InternetGateways":
				formatInternetGatewaysSummary(&b, rep)
				// Add cases for other services
			}
		}
	}

	// Fallback case – no inventory found
	allEmpty := len(rep.Status.ContainerImages) == 0
	for _, svc := range monitoredServices {
		if svc.GetCount(rep) > 0 {
			allEmpty = false
			break
		}
	}

	if allEmpty {
		b.WriteString(
			fmt.Sprintf("\n_No inventory results found for `%s`._\n",
				rep.Spec.SourceRef.Name,
			),
		)
	}

	return b.String()
}

// collectTagKeys collects all unique tag keys from a list of tagged resources
func collectTagKeys(resourceTags []map[string]string) []string {
	tagKeys := make(map[string]bool)
	for _, tags := range resourceTags {
		for key := range tags {
			tagKeys[key] = true
		}
	}

	// Convert to sorted slice for consistent column order
	sortedTagKeys := make([]string, 0, len(tagKeys))
	for key := range tagKeys {
		sortedTagKeys = append(sortedTagKeys, key)
	}
	sort.Strings(sortedTagKeys)
	return sortedTagKeys
}

// writeCSVHeaderWithTags writes a CSV header line with base columns and tag columns
func writeCSVHeaderWithTags(b *strings.Builder, baseColumns string, tagKeys []string) {
	b.WriteString(baseColumns)
	for _, tagKey := range tagKeys {
		b.WriteString(fmt.Sprintf(",%s", tagKey))
	}
	b.WriteString("\n")
}

// writeCSVRowWithTags writes a row with tag values in their corresponding columns
func writeCSVRowWithTags(b *strings.Builder, baseRow string, tags map[string]string, tagKeys []string) {
	b.WriteString(baseRow)

	// Add values for each tag column (or empty string if tag doesn't exist)
	for _, tagKey := range tagKeys {
		tagValue, exists := tags[tagKey]
		if exists {
			// Escape commas in tag values to prevent breaking CSV format
			escapedValue := strings.ReplaceAll(tagValue, ",", "\\,")
			b.WriteString(fmt.Sprintf(",%s", escapedValue))
		} else {
			b.WriteString(",")
		}
	}
	b.WriteString("\n")
}

// formatEC2Summary formats the EC2 instances summary section
func formatEC2Summary(b *strings.Builder, rep *v1alpha1.CloudInventoryReport) {
	// Count instances by state
	total := len(rep.Status.EC2)
	stateCounts := make(map[string]int)
	for _, inst := range rep.Status.EC2 {
		state := strings.ToLower(inst.State)
		stateCounts[state]++
	}

	// Write summary section
	b.WriteString("#### Summary\n")
	b.WriteString(fmt.Sprintf("- Total Instances: `%d`\n", total))
	if count, ok := stateCounts["running"]; ok {
		b.WriteString(fmt.Sprintf("- Running: `%d`\n", count))
	}
	if count, ok := stateCounts["stopped"]; ok {
		b.WriteString(fmt.Sprintf("- Stopped: `%d`\n", count))
	}
	for state, count := range stateCounts {
		if state != "running" && state != "stopped" {
			b.WriteString(fmt.Sprintf("- %s: `%d`\n", cases.Title(language.English).String(state), count))
		}
	}

	// Collect tag data
	var allEC2Tags []map[string]string
	for _, inst := range rep.Status.EC2 {
		allEC2Tags = append(allEC2Tags, inst.Tags)
	}
	sortedTagKeys := collectTagKeys(allEC2Tags)

	// Write CSV summary
	b.WriteString("\n#### CSV Summary\n```csv\n")

	// Create headers with tag columns
	writeCSVHeaderWithTags(b, "Name,InstanceID,State,Type,AvailabilityZone,Platform,PublicIP,PrivateDNS,PrivateIP,ImageID,VPCID", sortedTagKeys)

	// Add each instance as a row
	for _, inst := range rep.Status.EC2 {
		// Format the base row data
		baseRow := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s",
			inst.Name, inst.InstanceID, inst.State, inst.Type, inst.AZ, inst.Platform,
			inst.PublicIP, inst.PrivateDNS, inst.PrivateIP, inst.ImageID, inst.VPCID)

		writeCSVRowWithTags(b, baseRow, inst.Tags, sortedTagKeys)
	}

	b.WriteString("```\n")
}

// formatRDSSummary formats the RDS instances summary section
func formatRDSSummary(b *strings.Builder, rep *v1alpha1.CloudInventoryReport) {
	engineCounts := make(map[string]int)
	statusCounts := make(map[string]int)
	for _, db := range rep.Status.RDS {
		engineCounts[strings.ToLower(db.Engine)]++
		statusCounts[strings.ToLower(db.Status)]++
	}

	b.WriteString("#### Summary\n")
	b.WriteString(fmt.Sprintf("- Total DB Instances: `%d`\n", len(rep.Status.RDS)))
	for engine, count := range engineCounts {
		b.WriteString(fmt.Sprintf("- Engine: `%s` → `%d`\n", engine, count))
	}
	for status, count := range statusCounts {
		b.WriteString(fmt.Sprintf("- Status: `%s` → `%d`\n", status, count))
	}

	// Collect RDS tag data if available
	var allRDSTags []map[string]string
	for _, db := range rep.Status.RDS {
		allRDSTags = append(allRDSTags, db.Tags)
	}

	// CSV output
	b.WriteString("\n#### CSV Summary\n```csv\n")

	baseColumns := "Identifier,Engine,Version,Class,AZ,Status,MultiAZ,Public,StorageType,Allocated,VPCID"
	if len(allRDSTags) > 0 {
		sortedTagKeys := collectTagKeys(allRDSTags)
		writeCSVHeaderWithTags(b, baseColumns, sortedTagKeys)

		for _, db := range rep.Status.RDS {
			baseRow := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%t,%t,%s,%d,%s",
				db.DBInstanceIdentifier, db.Engine, db.EngineVersion, db.InstanceClass,
				db.AvailabilityZone, db.Status, db.MultiAZ, db.PubliclyAccessible,
				db.StorageType, db.AllocatedStorage, db.VPCID)

			writeCSVRowWithTags(b, baseRow, db.Tags, sortedTagKeys)
		}
	} else {
		// No tags available, use simple format
		b.WriteString(baseColumns + "\n")
		for _, db := range rep.Status.RDS {
			b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%t,%t,%s,%d,%s\n",
				db.DBInstanceIdentifier, db.Engine, db.EngineVersion, db.InstanceClass,
				db.AvailabilityZone, db.Status, db.MultiAZ, db.PubliclyAccessible,
				db.StorageType, db.AllocatedStorage, db.VPCID))
		}
	}

	b.WriteString("```\n")
}

// formatELBV2Summary formats the ELBv2 summary section
func formatELBV2Summary(b *strings.Builder, rep *v1alpha1.CloudInventoryReport) {
	b.WriteString("#### Summary\n")
	b.WriteString(fmt.Sprintf("- Total Load Balancers: `%d`\n", len(rep.Status.ELBV2)))

	// Count load balancers by scheme
	schemeCounts := make(map[string]int)
	for _, lb := range rep.Status.ELBV2 {
		scheme := strings.ToLower(lb.Scheme)
		schemeCounts[scheme]++
	}
	for scheme, count := range schemeCounts {
		b.WriteString(fmt.Sprintf("- Scheme: `%s` → `%d`\n", scheme, count))
	}
	// Count load balancers by type
	typeCounts := make(map[string]int)
	for _, lb := range rep.Status.ELBV2 {
		lbType := strings.ToLower(lb.Type)
		typeCounts[lbType]++
	}
	for lbType, count := range typeCounts {
		b.WriteString(fmt.Sprintf("- Type: `%s` → `%d`\n", lbType, count))
	}

	// Collect ELB tag data
	var allELBTags []map[string]string
	for _, lb := range rep.Status.ELBV2 {
		allELBTags = append(allELBTags, lb.Tags)
	}

	b.WriteString("\n#### CSV Summary\n```csv\n")

	// If we have tags, use them
	baseColumns := "Name,ARN,DNSName,Scheme,Type,VPCID,State,IPAddressType,SecurityGroups,Subnets"
	if len(allELBTags) > 0 {
		sortedTagKeys := collectTagKeys(allELBTags)
		writeCSVHeaderWithTags(b, baseColumns, sortedTagKeys)

		for _, lb := range rep.Status.ELBV2 {
			baseRow := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%v,%v",
				lb.Name, lb.ARN, lb.DNSName, lb.Scheme,
				lb.Type, lb.VPCID, lb.State, lb.IPAddressType,
				strings.Join(lb.SecurityGroups, ","), strings.Join(lb.Subnets, ","))

			writeCSVRowWithTags(b, baseRow, lb.Tags, sortedTagKeys)
		}
	} else {
		// No tags available, use simple format
		b.WriteString(baseColumns + "\n")
		for _, lb := range rep.Status.ELBV2 {
			b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%v,%v\n",
				lb.Name, lb.ARN, lb.DNSName, lb.Scheme,
				lb.Type, lb.VPCID, lb.State, lb.IPAddressType,
				strings.Join(lb.SecurityGroups, ","), strings.Join(lb.Subnets, ",")))
		}
	}

	b.WriteString("```\n")
}

func formatS3Summary(b *strings.Builder, rep *v1alpha1.CloudInventoryReport) {
	b.WriteString("#### Summary\n")
	b.WriteString(fmt.Sprintf("- Total Buckets: `%d`\n", len(rep.Status.S3)))
	b.WriteString(fmt.Sprintf("- Potentially Public Buckets: `%d`\n",
		func() int {
			n := 0
			for _, b := range rep.Status.S3 {
				if b.BlockAllPublicAccess {
					n++
				}
			}
			return n
		}(),
	))

	for _, bucket := range rep.Status.S3 {
		b.WriteString(fmt.Sprintf("- `%s` (region: `%s`, BlockAllPublicAccess: `%t`)\n",
			bucket.Name, bucket.Region, bucket.BlockAllPublicAccess))
	}

	b.WriteString("\n#### CSV Summary\n```csv\n")
	b.WriteString("name,region,BlockAllPublicAccess,")

	tagCols := collectTagKeys(func() []map[string]string {
		var all []map[string]string
		for _, b := range rep.Status.S3 {
			all = append(all, b.Tags)
		}
		return all
	}())
	b.WriteString(strings.Join(tagCols, ",") + "\n")
	for _, bucket := range rep.Status.S3 {
		row := fmt.Sprintf("%s,%s,%t", bucket.Name, bucket.Region, bucket.BlockAllPublicAccess)
		for _, key := range tagCols {
			row += "," + bucket.Tags[key]
		}
		b.WriteString(row + "\n")
	}
	b.WriteString("```\n")
}

func formatEIPSummary(b *strings.Builder, rep *v1alpha1.CloudInventoryReport) {
	total := len(rep.Status.EIP)
	assoc := 0
	for _, e := range rep.Status.EIP {
		if e.InstanceID != "" || e.NetworkInterfaceID != "" {
			assoc++
		}
	}
	b.WriteString("#### Summary\n")
	b.WriteString(fmt.Sprintf("- Total Elastic IPs: `%d`\n", total))
	b.WriteString(fmt.Sprintf("- Associated: `%d`\n", assoc))
	b.WriteString(fmt.Sprintf("- Unassociated: `%d`\n", total-assoc))

	b.WriteString("\n#### CSV Summary\n```csv\n")
	b.WriteString("allocationId,publicIp,domain,instanceId,networkInterfaceId,privateIp\n")
	for _, e := range rep.Status.EIP {
		b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			e.AllocationID, e.PublicIP, e.Domain,
			e.InstanceID, e.NetworkInterfaceID, e.PrivateIP))
	}
	b.WriteString("```\n")
}

func formatECRSummary(b *strings.Builder, rep *v1alpha1.CloudInventoryReport) {
	b.WriteString("#### Summary\n")
	b.WriteString(fmt.Sprintf("- Total Repositories: `%d`\n", len(rep.Status.ECR)))

	b.WriteString("\n#### Repositories & Latest Images:\n")
	for _, e := range rep.Status.ECR {
		b.WriteString(fmt.Sprintf("- `%s` → `%s`\n", e.RepositoryName, e.LatestImageTag))
	}

	b.WriteString("\n#### CSV Summary\n```csv\n")
	b.WriteString("registryId,repositoryName,latestImageTag,latestImageDigest\n")
	for _, e := range rep.Status.ECR {
		b.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			e.RegistryID, e.RepositoryName, e.LatestImageTag, e.LatestImageDigest))
	}
	b.WriteString("```\n")
}

func formatNATGatewaysSummary(b *strings.Builder, rep *v1alpha1.CloudInventoryReport) {
	total := len(rep.Status.NATGateways)
	b.WriteString("#### Summary\n")
	b.WriteString(fmt.Sprintf("- Total NAT Gateways: `%d`\n", total))

	b.WriteString("\n#### CSV Summary\n```csv\n")
	b.WriteString("natGatewayId,vpcId,subnetId,state\n")
	for _, nat := range rep.Status.NATGateways {
		b.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			nat.NatGatewayId, nat.VpcId, nat.SubnetId, nat.State))
	}
	b.WriteString("```\n")
}

func formatInternetGatewaysSummary(b *strings.Builder, rep *v1alpha1.CloudInventoryReport) {
	total := len(rep.Status.InternetGateways)
	b.WriteString("#### Summary\n")
	b.WriteString(fmt.Sprintf("- Total Internet Gateways: `%d`\n", total))

	b.WriteString("\n#### CSV Summary\n```csv\n")
	b.WriteString("internetGatewayId,attachments\n")
	for _, ig := range rep.Status.InternetGateways {
		b.WriteString(fmt.Sprintf("%s,%s\n",
			ig.InternetGatewayId,
			strings.Join(ig.Attachments, "|"),
		))
	}
	b.WriteString("```\n")
}
