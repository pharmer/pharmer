package inspector

import (
	"fmt"

	term "github.com/appscode/go-term"
	"github.com/appscode/go/errors"
	"github.com/cenkalti/backoff"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (i *Inspector) CheckHelthStatus() error {
	term.Println("Checking for component status...")
	backoff.Retry(func() error {
		resp, err := i.client.CoreV1().ComponentStatuses().List(metav1.ListOptions{
			LabelSelector: labels.Everything().String(),
		})
		if err != nil {
			return err
		}
		for _, status := range resp.Items {
			for _, cond := range status.Conditions {
				if cond.Type == apiv1.ComponentHealthy && cond.Status != apiv1.ConditionTrue {
					return errors.New().WithMessagef("Component %v is in condition %v with status %v", status.Name, cond.Type, cond.Status).Err()
				} else {
					term.Infoln(fmt.Sprintf("Component %v is in condition %v with status %v", status.Name, cond.Type, cond.Status))
				}
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	term.Successln("Component status are ok")
	return nil
}

func (i *Inspector) checkRBAC() error {
	if _, err := i.client.Discovery().ServerResourcesForGroupVersion("authentication.k8s.io/v1beta1"); err != nil {
		term.Errorln("RBAC authentication is not enabled")
		return errors.FromErr(err).Err()
	} else {
		term.Successln("RBAC authentication is enabled")
	}
	if _, err := i.client.Discovery().ServerResourcesForGroupVersion("authorization.k8s.io/v1beta1"); err != nil {
		term.Errorln("RBAC authorization is not enabled")
		return errors.FromErr(err).Err()
	} else {
		term.Successln("RBAC authorization is enabled")
	}
	return nil
}
