// Code generated by protoc-gen-grpc-gateway-cors
// source: artifact.proto
// DO NOT EDIT!

/*
Package v1beta1 is a reverse proxy.

It translates gRPC into RESTful JSON APIs.
*/
package v1beta1

import "github.com/grpc-ecosystem/grpc-gateway/runtime"

// ExportArtifactsCorsPatterns returns an array of grpc gatway mux patterns for
// Artifacts service to enable CORS.
func ExportArtifactsCorsPatterns() []runtime.Pattern {
	return []runtime.Pattern{
		pattern_Artifacts_Search_0,
		pattern_Artifacts_List_0,
	}
}
