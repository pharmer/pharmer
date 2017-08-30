package cloud

import (
	"context"
	"net"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"k8s.io/apimachinery/pkg/util/wait"
)

func EnsureARecord(ctx context.Context, cluster *api.Cluster, master *api.Instance) error {
	clusterDomain := Extra(ctx).Domain(cluster.Name)
	// TODO: FixIT!
	//for _, ip := range system.Config.Compass.IPs {
	//	if err := DNSProvider(ctx).EnsureARecord(clusterDomain, ip); err != nil {
	//		return err
	//	}
	//}
	Logger(ctx).Infof("Cluster apps A record %v added", clusterDomain)
	externalDomain := Extra(ctx).ExternalDomain(cluster.Name)
	if err := DNSProvider(ctx).EnsureARecord(externalDomain, master.Status.ExternalIP); err != nil {
		return err
	}
	Logger(ctx).Infof("External A record %v added", externalDomain)
	internalDomain := Extra(ctx).InternalDomain(cluster.Name)
	if err := DNSProvider(ctx).EnsureARecord(internalDomain, master.Status.InternalIP); err != nil {
		return err
	}
	Logger(ctx).Infof("Internal A record %v added", internalDomain)
	return nil
}

func DeleteARecords(ctx context.Context, cluster *api.Cluster) error {
	clusterDomain := Extra(ctx).Domain(cluster.Name)
	if err := DNSProvider(ctx).DeleteARecords(clusterDomain); err == nil {
		Logger(ctx).Infof("Cluster apps A record %v deleted", clusterDomain)
	}

	externalDomain := Extra(ctx).ExternalDomain(cluster.Name)
	if err := DNSProvider(ctx).DeleteARecords(externalDomain); err == nil {
		Logger(ctx).Infof("External A record %v deleted", externalDomain)
	}

	internalDomain := Extra(ctx).InternalDomain(cluster.Name)
	if err := DNSProvider(ctx).DeleteARecords(internalDomain); err == nil {
		Logger(ctx).Infof("Internal A record %v deleted", internalDomain)
	}

	return nil
}

func EnsureDnsIPLookup(ctx context.Context, cluster *api.Cluster) error {
	if externalDomain := Extra(ctx).ExternalDomain(cluster.Name); externalDomain != "" {
		err := wait.Poll(30*time.Second, 10*time.Minute, func() (bool, error) {
			Logger(ctx).Infof("Verifying external DNS %v ... ", externalDomain)
			ips, err := net.LookupIP(externalDomain)
			return len(ips) > 0, err
		})
		if err != nil {
			return errors.New("External master DNS failed to propagate in allocated time slot").WithContext(ctx).Err()
		}
	}
	if internalDomain := Extra(ctx).InternalDomain(cluster.Name); internalDomain != "" {
		err := wait.Poll(30*time.Second, 10*time.Minute, func() (bool, error) {
			Logger(ctx).Infof("Verifying internal DNS %v ...", internalDomain)
			ips, err := net.LookupIP(internalDomain)
			return len(ips) > 0, err
		})
		if err != nil {
			return errors.New("Internal master DNS failed to propagate in allocated time slot").WithContext(ctx).Err()
		}
	}
	return nil
}
