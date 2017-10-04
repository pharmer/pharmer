package cloud

import (
	"fmt"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	bootstrapapi "k8s.io/kubernetes/pkg/bootstrap/api"
)

var (
	TypeBootstrapToken           = "bootstrap.kubernetes.io/token"
	DefaultTokenUsages           = []string{"signing", "authentication"}
	BootstrapGroupPattern        = "system:bootstrappers:[a-z0-9:-]{0,255}[a-z0-9]"
	BootstrapTokenExtraGroupsKey = "auth-extra-groups"
)

func CheckValidToken(kc kubernetes.Interface) {
	secrets, err := kc.CoreV1().Secrets(metav1.NamespaceSystem).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(map[string]string{
			"type": TypeBootstrapToken,
		}).String(),
	})
	if err != nil {
		fmt.Println(err)
	}
	for _, secret := range secrets.Items {
		data := secret.Data["expiration"]
		fmt.Println(string(data))
	}
}

func CreateValidToken(kc kubernetes.Interface) error {
	tokenID, tokenSecret, err := ParseToken(GetKubeadmToken())
	if err != nil {
		return err
	}
	secretName := fmt.Sprintf("%s%s", bootstrapapi.BootstrapTokenSecretPrefix, tokenID)
	description := "Bootstrap token generated for 24 hours"
	tokenDuration := 24 * time.Hour
	usages := DefaultTokenUsages
	extraGroups := []string{"system:bootstrappers:kubeadm:default-node-token"}
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: apiv1.SecretType(bootstrapapi.SecretTypeBootstrapToken),
		Data: encodeTokenSecretData(tokenID, tokenSecret, tokenDuration, usages, extraGroups, description),
	}

	if _, err := kc.CoreV1().Secrets(metav1.NamespaceSystem).Create(secret); err != nil {
		return err
	}
	return nil
}

// encodeTokenSecretData takes the token discovery object and an optional duration and returns the .Data for the Secret
func encodeTokenSecretData(tokenID, tokenSecret string, duration time.Duration, usages []string, extraGroups []string, description string) map[string][]byte {
	data := map[string][]byte{
		bootstrapapi.BootstrapTokenIDKey:     []byte(tokenID),
		bootstrapapi.BootstrapTokenSecretKey: []byte(tokenSecret),
	}

	if len(extraGroups) > 0 {
		data[BootstrapTokenExtraGroupsKey] = []byte(strings.Join(extraGroups, ","))
	}

	if duration > 0 {
		// Get the current time, add the specified duration, and format it accordingly
		durationString := time.Now().Add(duration).Format(time.RFC3339)
		data[bootstrapapi.BootstrapTokenExpirationKey] = []byte(durationString)
	}
	if len(description) > 0 {
		data[bootstrapapi.BootstrapTokenDescriptionKey] = []byte(description)
	}
	for _, usage := range usages {
		// TODO: Validate the usage string here before
		data[bootstrapapi.BootstrapTokenUsagePrefix+usage] = []byte("true")
	}
	return data
}
