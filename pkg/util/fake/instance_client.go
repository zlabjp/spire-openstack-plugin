package fake

import (
	"errors"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"github.com/zlabjp/spire-openstack-plugin/pkg/openstack"
)

type Instance struct {
	projectID string
	metaData  map[string]string
	secGroup  []map[string]interface{}
	created   time.Time
}

// NewInstance returns fake InstanceClient which returns data including given projectID
func NewInstance(projectID string, metaData map[string]string, secGroup []map[string]interface{}) openstack.InstanceClient {
	return &Instance{
		projectID: projectID,
		metaData:  metaData,
		secGroup:  secGroup,
		created:   time.Now(),
	}
}

func NewInstanceWithTime(projectID string, created time.Time) openstack.InstanceClient {
	return &Instance{
		projectID: projectID,
		created:   created,
	}
}

func (f *Instance) Get(uuid string) (*servers.Server, error) {
	return &servers.Server{
		ID:             uuid,
		Name:           "bravo",
		TenantID:       f.projectID,
		Addresses:      map[string]interface{}{},
		Metadata:       f.metaData,
		SecurityGroups: f.secGroup,
		Created:        f.created,
		Updated:        f.created,
	}, nil
}

type ErrorInstance struct {
	message string
}

// NewErrorInstance returns ErrorInstance which always returns error
func NewErrorInstance(msg string) openstack.InstanceClient {
	return &ErrorInstance{
		message: msg,
	}
}

func (f *ErrorInstance) Get(uuid string) (*servers.Server, error) {
	return nil, errors.New(f.message)
}
