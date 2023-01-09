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
	minute string
	hour   string
	day    string
	month  string
	year   string
}

type EcsFargateTaskDetail struct {
	// cpuArchitecture string
	// os              string
	cpu         float64
	memoryInMiB float64
	clusterName string
	clusterArn  string
	defaultVpc  bool
	// Vpc         struct {
	// 	id                        string
	// 	cidr                      string
	// 	availablityZones          []string
	// 	publicSubnetIds           []string
	// 	publicSubnetRouteTableIds []string
	// }

	Container struct {
		name                string
		isEssential         bool
		ecrImageNameWithTag string
		logPrefix           string
		environmentVariable map[string]string
	}

	policies iam.PolicyDocument
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
			*jsii.String("DefaultPolicy"): props.policies,
		},
	})

	fargateTaskDef := ecs.NewFargateTaskDefinition(this, jsii.String("EcsFargateTaskDef"), &ecs.FargateTaskDefinitionProps{
		Cpu:            jsii.Number(props.cpu),
		MemoryLimitMiB: jsii.Number(float64(props.memoryInMiB)),
		RuntimePlatform: &ecs.RuntimePlatform{
			CpuArchitecture:       ecs.CpuArchitecture_X86_64,
			OperatingSystemFamily: ecs.OperatingSystemFamily_LINUX,
		},
		TaskRole: taskRole,
	})

	logGroup := logs.LogGroup_FromLogGroupArn(this, jsii.String("LogGroup"), jsii.String(props.LogGroupArn))
	ecrImageNameTagSplit := strings.Split(props.Container.ecrImageNameWithTag, ":")
	envVars := make(map[string]*string)
	for key, value := range props.Container.environmentVariable {
		envVars[key] = jsii.String(value)
	}
	ecs.NewContainerDefinition(this, jsii.String("ContainerDef"), &ecs.ContainerDefinitionProps{
		ContainerName: &props.Container.name,
		Essential:     jsii.Bool(props.Container.isEssential),
		Image: ecs.AssetImage_FromEcrRepository(
			ecr.Repository_FromRepositoryName(this, jsii.String("EcrRepo"), jsii.String(ecrImageNameTagSplit[0])), jsii.String(ecrImageNameTagSplit[1])),
		TaskDefinition: fargateTaskDef,
		Logging: ecs.AwsLogDriver_AwsLogs(&ecs.AwsLogDriverProps{
			LogGroup: logGroup,
		}),
		Environment: &envVars,
	})

	events.NewRule(this, jsii.String("EventsRule"), &events.RuleProps{
		Enabled: jsii.Bool(true),
		Schedule: events.Schedule_Cron(
			&events.CronOptions{
				Minute: &props.minute,
				Hour:   &props.hour,
				Day:    &props.day,
				Month:  &props.month,
				Year:   &props.year,
			},
		),
		Targets: &[]events.IRuleTarget{
			eventstargets.NewEcsTask(&eventstargets.EcsTaskProps{
				Cluster: ecs.Cluster_FromClusterAttributes(this,
					jsii.String("EcsTaskCluster"), &ecs.ClusterAttributes{
						ClusterName: jsii.String(props.clusterName),
						Vpc: ec2.Vpc_FromLookup(this, jsii.String("ClusterVpc"), &ec2.VpcLookupOptions{
							IsDefault: jsii.Bool(true),
						}),
					}),
			}),
		},
	})

	return &cronEcsFargateTask{this}

}
