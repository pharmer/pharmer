package cloud

import (
	"fmt"
	"strings"
	"time"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

const tokenCreateRetries = 5

func GetExistingKubeadmToken(kc kubernetes.Interface, duration time.Duration) (string, error) {
	for i := 0; i < tokenCreateRetries; i++ {
		secrets, err := kc.CoreV1().Secrets(metav1.NamespaceSystem).List(metav1.ListOptions{
			FieldSelector: fields.SelectorFromSet(map[string]string{
				"type": string(bootstrapapi.SecretTypeBootstrapToken),
			}).String(),
		})
		if err != nil {
			return "", err
		}
		now := time.Now()
		now.Format(time.RFC3339)
		for _, secret := range secrets.Items {
			data := secret.Data[bootstrapapi.BootstrapTokenExpirationKey]
			t, _ := time.Parse(time.RFC3339, string(data))
			if now.Before(t.Add(-60 * time.Minute)) { // at least valid for 60 mins
				return decodeToken(secret.Data[bootstrapapi.BootstrapTokenIDKey], secret.Data[bootstrapapi.BootstrapTokenSecretKey]), nil
			}
		}
		time.Sleep(15 * time.Second)
	}
	return CreateValidKubeadmToken(kc, duration)
}

func CreateValidKubeadmToken(kc kubernetes.Interface, duration time.Duration) (string, error) {
	token := GetKubeadmToken()
	tokenID, tokenSecret, err := ParseToken(token)
	if err != nil {
		return "", err
	}
	secretName := fmt.Sprintf("%s%s", bootstrapapi.BootstrapTokenSecretPrefix, tokenID)
	description := "Bootstrap token generated for 24 hours"
	extraGroups := []string{kubeadmconsts.NodeBootstrapTokenAuthGroup}
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: bootstrapapi.SecretTypeBootstrapToken,
		Data: encodeTokenSecretData(tokenID, tokenSecret, duration, kubeadmconsts.DefaultTokenUsages, extraGroups, description),
	}

	if _, err := kc.CoreV1().Secrets(metav1.NamespaceSystem).Create(secret); err != nil {
		return "", err
	}
	return token, nil
}

// encodeTokenSecretData takes the token discovery object and an optional duration and returns the .Data for the Secret
func encodeTokenSecretData(tokenID, tokenSecret string, duration time.Duration, usages []string, extraGroups []string, description string) map[string][]byte {
	data := map[string][]byte{
		bootstrapapi.BootstrapTokenIDKey:     []byte(tokenID),
		bootstrapapi.BootstrapTokenSecretKey: []byte(tokenSecret),
	}

	if len(extraGroups) > 0 {
		data[bootstrapapi.BootstrapTokenExtraGroupsKey] = []byte(strings.Join(extraGroups, ","))
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

func decodeToken(tokenID, tokenSecret []byte) string {
	return string(tokenID) + "." + string(tokenSecret)
}
