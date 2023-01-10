package golangdemoapp

import (
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	ec2 "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	ecr "github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	ecs "github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	events "github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	eventstargets "github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	iam "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	logs "github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CronExpression struct {
	Minute string
	Hour   string
	Day    string
	Month  string
	Year   string
}

type EcsFargateTaskDetail struct {
	// cpuArchitecture string
	// os              string
	Cpu             float64
	MemoryInMiB     float64
	ClusterName     string
	ClusterArn      string
	DefaultVpc      bool
	TaskPolicies    iam.PolicyDocument
	ContainerDetail Container
	// Vpc         struct {
	// 	id                        string
	// 	cidr                      string
	// 	availablityZones          []string
	// 	publicSubnetIds           []string
	// 	publicSubnetRouteTableIds []string
	// }

}

type Container struct {
	Name                 string
	IsEssential          bool
	EecrImageNameWithTag string
	LogPrefix            string
	EnvironmentVariable  map[string]string
}

type CronEcsFargateTaskProps struct {
	CronExpression
	EcsFargateTaskDetail
	LogGroupName string
	LogGroupArn  string
}

type cronEcsFargateTask struct {
	constructs.Construct
}

type CronEcsFargateTask interface {
	constructs.Construct
}

func NewCronEcsFargateTask(scope constructs.Construct, id string, props *CronEcsFargateTaskProps) CronEcsFargateTask {
	this := constructs.NewConstruct(scope, &id)

	taskRole := iam.NewRole(this, jsii.String("TaskRole"), &iam.RoleProps{
		AssumedBy: iam.NewServicePrincipal(jsii.String("ecs-tasks."+*awscdk.Aws_URL_SUFFIX()), nil),
		InlinePolicies: &map[string]iam.PolicyDocument{
			*jsii.String("DefaultPolicy"): props.TaskPolicies,
		},
	})

	fargateTaskDef := ecs.NewFargateTaskDefinition(this, jsii.String("EcsFargateTaskDef"), &ecs.FargateTaskDefinitionProps{
		Cpu:            jsii.Number(props.Cpu),
		MemoryLimitMiB: jsii.Number(props.MemoryInMiB),
		// RuntimePlatform: &ecs.RuntimePlatform{
		// 	// CpuArchitecture:       ecs.CpuArchitecture_X86_64,
		// 	OperatingSystemFamily: ecs.OperatingSystemFamily_LINUX,
		// },
		TaskRole: taskRole,
	})

	logGroup := logs.LogGroup_FromLogGroupArn(this, jsii.String("LogGroup"), jsii.String(props.LogGroupArn))
	ecrImageNameTagSplit := strings.Split(props.ContainerDetail.EecrImageNameWithTag, ":")
	envVars := make(map[string]*string)
	for key, value := range props.ContainerDetail.EnvironmentVariable {
		envVars[key] = jsii.String(value)
	}
	ecs.NewContainerDefinition(this, jsii.String("ContainerDef"), &ecs.ContainerDefinitionProps{
		ContainerName: &props.ContainerDetail.Name,
		Essential:     jsii.Bool(props.ContainerDetail.IsEssential),
		Image: ecs.AssetImage_FromEcrRepository(
			ecr.Repository_FromRepositoryName(this, jsii.String("EcrRepo"), jsii.String(ecrImageNameTagSplit[0])), jsii.String(ecrImageNameTagSplit[1])),
		TaskDefinition: fargateTaskDef,
		Logging: ecs.AwsLogDriver_AwsLogs(&ecs.AwsLogDriverProps{
			LogGroup:     logGroup,
			StreamPrefix: &props.ContainerDetail.LogPrefix,
		}),
		Environment: &envVars,
	})

	events.NewRule(this, jsii.String("EventsRule"), &events.RuleProps{
		Enabled: jsii.Bool(true),
		Schedule: events.Schedule_Cron(
			&events.CronOptions{
				Minute: &props.Minute,
				Hour:   &props.Hour,
				Day:    &props.Day,
				Month:  &props.Month,
				Year:   &props.Year,
			},
		),
		Targets: &[]events.IRuleTarget{
			eventstargets.NewEcsTask(&eventstargets.EcsTaskProps{
				Cluster: ecs.Cluster_FromClusterAttributes(this,
					jsii.String("EcsTaskCluster"), &ecs.ClusterAttributes{
						ClusterName:    jsii.String(props.ClusterName),
						SecurityGroups: &[]ec2.ISecurityGroup{},
						Vpc: ec2.Vpc_FromLookup(this, jsii.String("ClusterVpc"), &ec2.VpcLookupOptions{
							IsDefault: jsii.Bool(true),
						}),
					}),
				SubnetSelection: &ec2.SubnetSelection{
					SubnetType: ec2.SubnetType_PUBLIC,
					OnePerAz:   jsii.Bool(true),
				},
				TaskDefinition: fargateTaskDef,
			}),
		},
	})

	return &cronEcsFargateTask{this}

}
