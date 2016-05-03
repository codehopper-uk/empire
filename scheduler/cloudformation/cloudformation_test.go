package cloudformation

import (
	"bytes"
	"database/sql"
	"errors"
	"html/template"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	_ "github.com/lib/pq"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestScheduler_Submit_NewStack(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{}, awserr.New("400", "Stack with id acme-inc does not exist", errors.New("")))

	c.On("CreateStack", &cloudformation.CreateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		Tags: []*cloudformation.Tag{
			{Key: aws.String("empire.app.id"), Value: aws.String("c9366591-ab68-4d49-a333-95ce5a23df68")},
			{Key: aws.String("empire.app.name"), Value: aws.String("acme-inc")},
		},
	}).Return(&cloudformation.CreateStackOutput{}, nil)

	c.On("WaitUntilStackCreateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	err := s.Submit(context.Background(), &scheduler.App{
		ID:   "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_ExistingStack(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil)

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	err := s.Submit(context.Background(), &scheduler.App{
		ID:   "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_StackUpdateInProgress(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("UPDATE_IN_PROGRESS")},
		},
	}, nil)

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil).Twice()

	err := s.Submit(context.Background(), &scheduler.App{
		ID:   "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Remove(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	c.On("DeleteStack", &cloudformation.DeleteStackInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DeleteStackOutput{}, nil)

	err = s.Remove(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68")
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Remove_NoStack(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
	}

	err := s.Remove(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68")
	assert.Equal(t, errNoStack, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Instances(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		s3:             x,
		ecs:            e,
		db:             db,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	c.On("ListStackResourcesPages", &cloudformation.ListStackResourcesInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.ListStackResourcesOutput{
		StackResourceSummaries: []*cloudformation.StackResourceSummary{
			{ResourceType: aws.String("AWS::EC2::LoadBalancer")},
			{ResourceType: aws.String("AWS::ECS::Service"), LogicalResourceId: aws.String("web")},
		},
	}, nil)

	c.On("DescribeStackResource", &cloudformation.DescribeStackResourceInput{
		LogicalResourceId: aws.String("web"),
		StackName:         aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStackResourceOutput{
		StackResourceDetail: &cloudformation.StackResourceDetail{
			Metadata:           aws.String(`{"name":"web"}`),
			PhysicalResourceId: aws.String(`arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web`),
		},
	}, nil)

	e.On("ListTasksPages", &ecs.ListTasksInput{
		Cluster:     aws.String("cluster"),
		ServiceName: aws.String("acme-inc-web"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
		},
	}, nil)

	e.On("ListTasksPages", &ecs.ListTasksInput{
		Cluster:   aws.String("cluster"),
		StartedBy: aws.String("c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
		},
	}, nil)

	dt := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	e.On("DescribeTasks", &ecs.DescribeTasksInput{
		Cluster: aws.String("cluster"),
		Tasks: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
		},
	}).Return(&ecs.DescribeTasksOutput{
		Tasks: []*ecs.Task{
			{
				TaskArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
				TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-web:0"),
				LastStatus:        aws.String("RUNNING"),
				StartedAt:         &dt,
			},
			{
				TaskArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
				TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc--run:0"),
				LastStatus:        aws.String("PENDING"),
				CreatedAt:         &dt,
			},
		},
	}, nil)

	e.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-web:0"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name:   aws.String("web"),
					Cpu:    aws.Int64(256),
					Memory: aws.Int64(int64(256)),
				},
			},
		},
	}, nil)

	e.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc--run:0"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name:   aws.String("run"),
					Cpu:    aws.Int64(256),
					Memory: aws.Int64(int64(256)),
				},
			},
		},
	}, nil)

	instances, err := s.Instances(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68")
	assert.NoError(t, err)
	assert.Equal(t, &scheduler.Instance{
		ID:        "0b69d5c0-d655-4695-98cd-5d2d526d9d5a",
		UpdatedAt: dt,
		State:     "RUNNING",
		Process: &scheduler.Process{
			Type:        "web",
			MemoryLimit: 256 * bytesize.MB,
			CPUShares:   256,
			Env:         make(map[string]string),
		},
	}, instances[0])
	assert.Equal(t, &scheduler.Instance{
		ID:        "c09f0188-7f87-4b0f-bfc3-16296622b6fe",
		UpdatedAt: dt,
		State:     "PENDING",
		Process: &scheduler.Process{
			Type:        "run",
			MemoryLimit: 256 * bytesize.MB,
			CPUShares:   256,
			Env:         make(map[string]string),
		},
	}, instances[1])

	c.AssertExpectations(t)
	x.AssertExpectations(t)
	e.AssertExpectations(t)
}

func newDB(t testing.TB) *sql.DB {
	db, err := sql.Open("postgres", "postgres://localhost/empire?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`TRUNCATE TABLE stacks`); err != nil {
		t.Fatal(err)
	}
	return db
}

type mockCloudFormationClient struct {
	cloudformationClient
	mock.Mock
}

func (m *mockCloudFormationClient) CreateStack(input *cloudformation.CreateStackInput) (*cloudformation.CreateStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.CreateStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) UpdateStack(input *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.UpdateStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) DeleteStack(input *cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DeleteStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) DescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DescribeStacksOutput), args.Error(1)
}

func (m *mockCloudFormationClient) ListStackResourcesPages(input *cloudformation.ListStackResourcesInput, fn func(*cloudformation.ListStackResourcesOutput, bool) bool) error {
	args := m.Called(input)
	fn(args.Get(0).(*cloudformation.ListStackResourcesOutput), true)
	return args.Error(1)
}

func (m *mockCloudFormationClient) DescribeStackResource(input *cloudformation.DescribeStackResourceInput) (*cloudformation.DescribeStackResourceOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DescribeStackResourceOutput), args.Error(1)
}

func (m *mockCloudFormationClient) WaitUntilStackCreateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

func (m *mockCloudFormationClient) WaitUntilStackUpdateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

type mockS3Client struct {
	mock.Mock
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

type mockECSClient struct {
	ecsClient
	mock.Mock
}

func (m *mockECSClient) ListTasksPages(input *ecs.ListTasksInput, fn func(p *ecs.ListTasksOutput, lastPage bool) (shouldContinue bool)) error {
	args := m.Called(input)
	fn(args.Get(0).(*ecs.ListTasksOutput), true)
	return args.Error(1)
}

func (m *mockECSClient) DescribeTasks(input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeTasksOutput), args.Error(1)
}

func (m *mockECSClient) DescribeTaskDefinition(input *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeTaskDefinitionOutput), args.Error(1)
}
