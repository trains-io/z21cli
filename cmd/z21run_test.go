package cmd

import (
	"testing"

	"github.com/trains-io/z21cli/internal/contextstore"
	"github.com/trains-io/z21.go/client"
)

func TestPlanZ21DialMaintenancePort(t *testing.T) {
	app := &AppContext{
		Client: &client.Client{},
		Context: contextstore.Info{
			Host: "192.168.0.100",
			Port: client.DefaultPort,
		},
	}

	plan, err := planZ21Dial("", client.MaintenancePort, app)
	if err != nil {
		t.Fatal(err)
	}
	if plan.useApp {
		t.Fatal("expected separate dial for maintenance port")
	}
	if plan.host != "192.168.0.100" || plan.port != client.MaintenancePort {
		t.Fatalf("plan = %+v, want 192.168.0.100:%d", plan, client.MaintenancePort)
	}
}

func TestPlanZ21DialReusesAppClient(t *testing.T) {
	app := &AppContext{
		Client: &client.Client{},
		Context: contextstore.Info{
			Host: "192.168.0.100",
			Port: client.DefaultPort,
		},
	}

	plan, err := planZ21Dial("", 0, app)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.useApp {
		t.Fatal("expected app client reuse on default port")
	}
}

func TestPlanZ21DialReusesAppClientCustomPort(t *testing.T) {
	const customPort = 21107

	app := &AppContext{
		Client: &client.Client{},
		Context: contextstore.Info{
			Host: "192.168.0.100",
			Port: customPort,
		},
	}

	plan, err := planZ21Dial("", 0, app)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.useApp {
		t.Fatal("expected app client reuse on custom context port")
	}
	if plan.port != customPort {
		t.Fatalf("plan.port = %d, want %d", plan.port, customPort)
	}
}

func TestPlanZ21DialDirectHostDefaultPort(t *testing.T) {
	plan, err := planZ21Dial("10.0.0.5", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.useApp || plan.host != "10.0.0.5" || plan.port != client.DefaultPort {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestPlanZ21DialDirectHost(t *testing.T) {
	plan, err := planZ21Dial("10.0.0.5", client.MaintenancePort, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.useApp || plan.host != "10.0.0.5" || plan.port != client.MaintenancePort {
		t.Fatalf("plan = %+v", plan)
	}
}
