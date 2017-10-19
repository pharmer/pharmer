package client

import (
	attic "github.com/appscode/api/attic/v1beta1"
	auth "github.com/appscode/api/auth/v1beta1"
	ca "github.com/appscode/api/certificate/v1beta1"
	ci "github.com/appscode/api/ci/v1beta1"
	kubernetesV1beta1 "github.com/appscode/api/kubernetes/v1beta1"
	namespace "github.com/appscode/api/namespace/v1beta1"
	"google.golang.org/grpc"
)

// multi client services are grouped by there main client. the api service
// clients are wrapped around with sub-service.
type multiClientInterface interface {
	Attic() *atticService
	Authentication() *authenticationService
	CA() *caService
	CI() *ciService
	Namespace() *nsService
	Kubernetes() *versionedKubernetesService
}

type multiClientServices struct {
	atticClient               *atticService
	authenticationClient      *authenticationService
	caClient                  *caService
	ciClient                  *ciService
	nsClient                  *nsService
	versionedKubernetesClient *versionedKubernetesService
}

func newMultiClientService(conn *grpc.ClientConn) multiClientInterface {
	return &multiClientServices{
		atticClient: &atticService{
			artifactClient: attic.NewArtifactsClient(conn),
			versionClient:  attic.NewVersionsClient(conn),
		},
		authenticationClient: &authenticationService{
			authenticationClient: auth.NewAuthenticationClient(conn),
			conduitClient:        auth.NewConduitClient(conn),
			projectClient:        auth.NewProjectsClient(conn),
		},
		caClient: &caService{
			certificateClient: ca.NewCertificatesClient(conn),
		},
		ciClient: &ciService{
			agentsClient:   ci.NewAgentsClient(conn),
			metadataClient: ci.NewMetadataClient(conn),
		},
		versionedKubernetesClient: &versionedKubernetesService{
			v1beta1Service: &kubernetesV1beta1Service{
				clusterClient:  kubernetesV1beta1.NewClustersClient(conn),
				incidentClient: kubernetesV1beta1.NewIncidentsClient(conn),
				metdataClient:  kubernetesV1beta1.NewMetadataClient(conn),
				clientsClient:  kubernetesV1beta1.NewClientsClient(conn),
				diskClient:     kubernetesV1beta1.NewDisksClient(conn),
			},
		},
		nsClient: &nsService{
			teamClient: namespace.NewTeamsClient(conn),
		},
	}
}

func (s *multiClientServices) Attic() *atticService {
	return s.atticClient
}

func (s *multiClientServices) Authentication() *authenticationService {
	return s.authenticationClient
}

func (s *multiClientServices) Namespace() *nsService {
	return s.nsClient
}

func (s *multiClientServices) CA() *caService {
	return s.caClient
}

func (s *multiClientServices) CI() *ciService {
	return s.ciClient
}

func (s *multiClientServices) Kubernetes() *versionedKubernetesService {
	return s.versionedKubernetesClient
}

// original service clients that needs to exposed under grouped wrapper
// services.
type atticService struct {
	artifactClient attic.ArtifactsClient
	versionClient  attic.VersionsClient
}

func (a *atticService) Artifacts() attic.ArtifactsClient {
	return a.artifactClient
}

func (a *atticService) Versions() attic.VersionsClient {
	return a.versionClient
}

type authenticationService struct {
	authenticationClient auth.AuthenticationClient
	conduitClient        auth.ConduitClient
	projectClient        auth.ProjectsClient
}

func (a *authenticationService) Authentication() auth.AuthenticationClient {
	return a.authenticationClient
}

func (a *authenticationService) Conduit() auth.ConduitClient {
	return a.conduitClient
}

func (a *authenticationService) Project() auth.ProjectsClient {
	return a.projectClient
}

type ciService struct {
	agentsClient   ci.AgentsClient
	metadataClient ci.MetadataClient
}

func (a *ciService) Agents() ci.AgentsClient {
	return a.agentsClient
}

func (a *ciService) Metadata() ci.MetadataClient {
	return a.metadataClient
}

type nsService struct {
	teamClient namespace.TeamsClient
}

func (b *nsService) Team() namespace.TeamsClient {
	return b.teamClient
}

type caService struct {
	certificateClient ca.CertificatesClient
}

func (c *caService) CertificatesClient() ca.CertificatesClient {
	return c.certificateClient
}

type versionedKubernetesService struct {
	v1beta1Service *kubernetesV1beta1Service
}

func (v *versionedKubernetesService) V1beta1() *kubernetesV1beta1Service {
	return v.v1beta1Service
}

type kubernetesV1beta1Service struct {
	clusterClient  kubernetesV1beta1.ClustersClient
	incidentClient kubernetesV1beta1.IncidentsClient
	metdataClient  kubernetesV1beta1.MetadataClient
	clientsClient  kubernetesV1beta1.ClientsClient
	diskClient     kubernetesV1beta1.DisksClient
}

func (k *kubernetesV1beta1Service) Cluster() kubernetesV1beta1.ClustersClient {
	return k.clusterClient
}

func (a *kubernetesV1beta1Service) Incident() kubernetesV1beta1.IncidentsClient {
	return a.incidentClient
}

func (k *kubernetesV1beta1Service) Metadata() kubernetesV1beta1.MetadataClient {
	return k.metdataClient
}

func (k *kubernetesV1beta1Service) Client() kubernetesV1beta1.ClientsClient {
	return k.clientsClient
}

func (k *kubernetesV1beta1Service) Disk() kubernetesV1beta1.DisksClient {
	return k.diskClient
}
