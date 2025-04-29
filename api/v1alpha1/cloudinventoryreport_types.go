package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EC2InstanceInfo struct {
	Name       string            `json:"name"`
	InstanceID string            `json:"instanceId"`
	State      string            `json:"state"`
	Type       string            `json:"type"`
	AZ         string            `json:"az"`
	Platform   string            `json:"platform"`
	PublicIP   string            `json:"publicIp"`
	PrivateDNS string            `json:"privateDns"`
	PrivateIP  string            `json:"privateIp"`
	ImageID    string            `json:"imageId"`
	VPCID      string            `json:"vpcId"`
	Tags       map[string]string `json:"tags"`
}

type RDSInstanceInfo struct {
	DBInstanceIdentifier string            `json:"dbInstanceIdentifier"`
	Engine               string            `json:"engine"`
	EngineVersion        string            `json:"engineVersion"`
	InstanceClass        string            `json:"instanceClass"`
	AvailabilityZone     string            `json:"availabilityZone"`
	Status               string            `json:"status"`
	MultiAZ              bool              `json:"multiAz"`
	PubliclyAccessible   bool              `json:"publiclyAccessible"`
	StorageType          string            `json:"storageType"`
	AllocatedStorage     int32             `json:"allocatedStorage"`
	VPCID                string            `json:"vpcId,omitempty"`
	Tags                 map[string]string `json:"tags"`
}

type ELBV2Info struct {
	Name           string            `json:"name" yaml:"name"`
	ARN            string            `json:"arn" yaml:"arn"`
	DNSName        string            `json:"dnsName" yaml:"dnsName"`
	Scheme         string            `json:"scheme" yaml:"scheme"`
	Type           string            `json:"type" yaml:"type"`
	VPCID          string            `json:"vpcId" yaml:"vpcId"`
	State          string            `json:"state" yaml:"state"`
	IPAddressType  string            `json:"ipAddressType" yaml:"ipAddressType"`
	SecurityGroups []string          `json:"securityGroups" yaml:"securityGroups"`
	Subnets        []string          `json:"subnets" yaml:"subnets"`
	Tags           map[string]string `json:"tags" yaml:"tags"`
}

type S3BucketInfo struct {
	Name                 string            `json:"name" yaml:"name"`
	Region               string            `json:"region" yaml:"region"`
	BlockAllPublicAccess bool              `json:"blockAllPublicAccess" yaml:"blockAllPublicAccess"`
	Tags                 map[string]string `json:"tags" yaml:"tags"`
}

type EIPInfo struct {
	AllocationID       string            `json:"allocationId" yaml:"allocationId"`
	PublicIP           string            `json:"publicIp" yaml:"publicIp"`
	Domain             string            `json:"domain" yaml:"domain"` // “vpc” or “standard”
	InstanceID         string            `json:"instanceId,omitempty" yaml:"instanceId"`
	NetworkInterfaceID string            `json:"networkInterfaceId,omitempty" yaml:"networkInterfaceId"`
	PrivateIP          string            `json:"privateIp,omitempty" yaml:"privateIp"`
	Tags               map[string]string `json:"tags,omitempty" yaml:"tags"`
}

type ECRRegistryInfo struct {
	RegistryID        string `json:"registryId"`
	RepositoryName    string `json:"repositoryName"`
	LatestImageTag    string `json:"latestImageTag"`
	LatestImageDigest string `json:"latestImageDigest,omitempty"`
}

type NATGatewayInfo struct {
	NatGatewayId string            `json:"natGatewayId" yaml:"natGatewayId"`
	VpcId        string            `json:"vpcId" yaml:"vpcId"`
	SubnetId     string            `json:"subnetId,omitempty" yaml:"subnetId,omitempty"`
	State        string            `json:"state,omitempty" yaml:"state,omitempty"`
	Tags         map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type InternetGatewayInfo struct {
	InternetGatewayId string            `json:"internetGatewayId" yaml:"internetGatewayId"`
	Attachments       []string          `json:"attachments,omitempty" yaml:"attachments,omitempty"`
	Tags              map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CloudInventoryReportSpec defines the desired state of CloudInventoryReport
type CloudInventoryReportSpec struct {
	SourceRef corev1.ObjectReference `json:"sourceRef"`
	Timestamp metav1.Time            `json:"timestamp"`
}

// CloudInventoryReportStatus defines the observed state of CloudInventoryReport
type CloudInventoryReportStatus struct {
	//AWS
	EC2              []EC2InstanceInfo     `json:"ec2,omitempty"`
	RDS              []RDSInstanceInfo     `json:"rds,omitempty"`
	ELBV2            []ELBV2Info           `json:"elbv2,omitempty"`
	S3               []S3BucketInfo        `json:"s3,omitempty"`
	EIP              []EIPInfo             `json:"eip,omitempty"`
	ECR              []ECRRegistryInfo     `json:"ecr,omitempty"`
	NATGateways      []NATGatewayInfo      `json:"natGateways,omitempty"`
	InternetGateways []InternetGatewayInfo `json:"internetGateways,omitempty"`

	// Kubernetes
	ContainerImages []ContainerImageInfo `json:"containerImages,omitempty" yaml:"containerImages,omitempty"`
	Summary         map[string]int       `json:"summary,omitempty" yaml:"summary,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CloudInventoryReport is the Schema for the cloudinventoryreports API
type CloudInventoryReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudInventoryReportSpec   `json:"spec,omitempty"`
	Status CloudInventoryReportStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudInventoryReportList contains a list of CloudInventoryReport
type CloudInventoryReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudInventoryReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudInventoryReport{}, &CloudInventoryReportList{})
}
