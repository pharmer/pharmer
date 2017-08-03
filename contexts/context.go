package contexts

import (
	"net"
	"strings"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"github.com/appscode/pharmer/contexts/auth"
	"github.com/appscode/pharmer/system"
	goContext "golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	elastic "gopkg.in/olivere/elastic.v3"
)

type context struct {
	Auth *auth.AuthInfo
	// Store             *storage.SqlStore
	BackgroundContext goContext.Context
	Metadata          map[string]interface{}
	Fake              bool
	logger            *log.Logger
}

func (c *context) Logger() *log.Logger {
	return c.logger
}

func NewContext(ctx goContext.Context) (*context, error) {
	md := ctx.Value(auth.Authentication)
	a, ok := md.(*auth.AuthInfo)
	if !ok {
		return nil, errors.New("no auth info found in context").Err()
	}
	c, err := newContextFromAuth(a)
	if err != nil {
		return nil, err
	}
	c.BackgroundContext = ctx
	c.Fake = false
	return c, nil
}

func newContextFromAuth(a *auth.AuthInfo) (*context, error) {
	if a == nil {
		return nil, errors.New("nil auth info found").Err()
	}
	//store, err := storage.NewByName(a.Namespace)
	//if err != nil {
	//	return nil, errors.FromErr(err).Err()
	//}
	ctx := &context{
		Auth: a,
		// Store:    store,
		Metadata: make(map[string]interface{}),
	}

	ctx.logger = ctx.newContextLogger()
	return ctx, nil
}

func NewBackgroundContext(a *auth.AuthInfo) goContext.Context {
	return goContext.WithValue(
		goContext.Background(),
		auth.Authentication,
		a,
	)
}

type elasticSearchContext struct {
	*context
	Client *elastic.Client
}

func newElasticSearchContext(ctx goContext.Context) (*elasticSearchContext, error) {
	c, err := NewContext(ctx)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}

	var es *elastic.Client
	if !c.Fake {
		es, err = elastic.NewClient(elastic.SetURL(system.Config.Artifactory.ElasticSearchEndpoint))
		if err != nil {
			return nil, errors.FromErr(err).Err()
		}
	}
	return &elasticSearchContext{
		Client:  es,
		context: c,
	}, nil
}

func GetClientIP(in goContext.Context) string {
	if pr, ok := peer.FromContext(in); ok {
		if pr.Addr != net.Addr(nil) {
			return pr.Addr.String()
		}
	}

	md, ok := metadata.FromIncomingContext(in)
	if ok {
		if ip, ok := md["X-Forwarded-For"]; ok {
			return strings.Join(ip, ",")
		}
	}
	// What is req.RemoteAddr
	return ""
}

func (c *context) String() string {
	return "[" + c.Auth.User.UserName + "@" + c.Auth.Namespace + "]"
}

func (c *context) newContextLogger() *log.Logger {
	return log.New(c)
}
