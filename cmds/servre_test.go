package cmds

import (
	"encoding/json"
	"fmt"
	"testing"

	api "code.gitea.io/sdk/gitea"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/pharmer/pharmer/apiserver"
)

func TestStan(t *testing.T) {
	conn, err := stan.Connect("pharmer-cluster", "te-0", stan.NatsURL("nats://localhost:4222"))

	fmt.Println(err, "^^^^^^^^^^^^^")
	defer apiserver.LogCloser(conn)

	op := &api.ClusterOperation{
		OperationID: "UQdbrzGzF91lYb5jFTnEIrvoBpnWp6",
	}
	data, err := json.Marshal(op)
	fmt.Println(string(data), err)

	err = conn.Publish("create-cluster", data)
	fmt.Println(err, "xxxxxxxxx")

}
