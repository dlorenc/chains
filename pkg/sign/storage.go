package sign

import (
	"context"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	_ "gocloud.dev/blob/gcsblob"
)

func SignStorage(ctx context.Context, s v1beta1.TaskResource, signer *Signer, tr *pipelinev1beta1.TaskRun, l *zap.SugaredLogger) error {
	rrs := map[string]string{}
	for _, rr := range tr.Status.ResourcesResult {
		if rr.ResourceRef.Name == s.Name {
			rrs[rr.Key] = rr.Value
		}
	}
	l.Info("rrs: %v", rrs)

	// Parse out the bucket name and object name from the URL
	url := rrs["url"]
	split := strings.Split(url, "#")
	gcsPath := split[0]
	// TODO: make sure it's gcs, not s3 or something
	objectPath := strings.TrimPrefix(gcsPath, "gs://")
	split = strings.Split(objectPath, "/")
	bucketName := split[0]
	object := strings.Join(split[1:], "/")

	// Now do object specific stuff.
	if strings.HasSuffix(gcsPath, ".txt") {
		// Example, read a .txt file and sign it
		// Upload the sig as <name>.txt.sig
		client, err := storage.NewClient(ctx)
		if err != nil {
			return err
		}
		bh := client.Bucket(bucketName)
		ob := bh.Object(object)
		blobReader, err := ob.NewReader(ctx)
		if err != nil {
			return err
		}

		signature, err := signer.SignReader(blobReader)
		if err != nil {
			return err
		}
		l.Info(string(signature))
		sigobj := bh.Object(object + ".sig")
		w := sigobj.NewWriter(ctx)
		defer w.Close()
		if _, err := w.Write(signature); err != nil {
			return err
		}
	} else if strings.HasSuffix(gcsPath, ".whl") {
		// Do pypi signature/upload here.
	}
	return nil
}
