// +build integration

package ots3

import (
	"context"
	"net/http"
	"testing"

	"github.com/DoNewsCode/core/key"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func setupManager() *Manager {
	return setupManagerWithTracer(nil)
}

func setupManagerWithTracer(tracer opentracing.Tracer) *Manager {
	m := NewManager(
		"Q3AM3UQ867SPQQA43P2F",
		"zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		"https://play.minio.io:9000",
		"asia",
		"mybucket",
		WithTracer(tracer),
	)
	_ = m.CreateBucket(context.Background(), "mybucket")
	return m
}

func TestNewManager(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewManager(
		"",
		"",
		"",
		"",
		"",
		WithTracer(opentracing.GlobalTracer()),
		WithPathPrefix(""),
		WithHttpClient(http.DefaultClient),
		WithLocationFunc(func(location string) (url string) {
			return ""
		}),
		WithKeyer(key.New()),
	))
}

func TestManager_CreateBucket(t *testing.T) {
	t.Parallel()
	m := NewManager("Q3AM3UQ867SPQQA43P2F", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG", "https://play.minio.io:9000", "asia", "mybucket")
	err := m.CreateBucket(context.Background(), "foo")
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyExists:
				return
			case s3.ErrCodeBucketAlreadyOwnedByYou:
				return
			default:
				t.Fail()
			}
		} else {
			t.Fail()
		}
		return
	}
}

func TestManager_UploadFromUrl(t *testing.T) {
	t.Parallel()
	tracer := mocktracer.New()
	m := setupManagerWithTracer(tracer)
	newURL, err := m.UploadFromUrl(context.Background(), "https://box.bdimg.com/static/fisp_static/common/img/searchbox/logo_news_276_88_1f9876a.png")
	assert.NoError(t, err)
	assert.NotEmpty(t, newURL)
	assert.NotEmpty(t, tracer.FinishedSpans())
}
