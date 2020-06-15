package sign

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dlorenc/chains/pkg/payload"
	"github.com/tektoncd/pipeline/pkg/version"

	// "strings"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

func AttachImageSignature(s v1beta1.TaskResource, signer *Signer, tr *pipelinev1beta1.TaskRun, l *zap.SugaredLogger) error {
	creds, err := k8schain.NewInCluster(k8schain.Options{
		Namespace:          "tekton-pipelines",
		ServiceAccountName: "tekton-pipelines-controller",
	})
	if err != nil {
		return err
	}

	rrs := map[string]string{}
	for _, rr := range tr.Status.ResourcesResult {
		if rr.ResourceRef.Name == s.Name {
			rrs[rr.Key] = rr.Value
		}
	}
	l.Info("rrs: %v", rrs)

	p := payload.CreatePayload(tr)

	sig := SimpleSigning{
		Critical: Critical{
			Identity: Identity{
				DockerReference: rrs["url"],
			},
			Image: Image{
				DockerManifestDigest: rrs["digest"],
			},
			Type: "Tekton builder signature",
		},
		Optional: map[string]interface{}{
			"builder":    fmt.Sprintf("Tekton %s", version.PipelineVersion),
			"provenance": tr.Status,
			"in_toto":    p,
		},
	}

	body, err := json.Marshal(sig)
	if err != nil {
		return err
	}

	l.Infof("Attaching signature %s to image %s", string(body), s.Name)

	signature, _, err := signer.Sign(sig)
	if err != nil {
		return err
	}

	tag, err := name.ParseReference(rrs["url"])
	if err != nil {
		return err
	}

	// orig, err := remote.Image(tag, remote.WithAuthFromKeychain(creds))
	// if err != nil {
	// 	return err
	// }

	// dgst, err := orig.Digest()
	// if err != nil {
	// 	return err
	// }

	dgst := rrs["digest"]

	signatureTar := bytes.Buffer{}
	w := tar.NewWriter(&signatureTar)
	w.WriteHeader(&tar.Header{
		Name: "signature",
		Size: int64(len(signature)),
		Mode: 0755,
	})
	w.Write(signature)
	w.WriteHeader(&tar.Header{
		Name: "body.json",
		Size: int64(len(body)),
		Mode: 0755,
	})
	w.Write(body)
	w.Close()

	// Now make the fake image to contain the signature object.
	layer, err := tarball.LayerFromReader(&signatureTar)
	if err != nil {
		return err
	}

	hex := strings.TrimPrefix(dgst, "sha256:")
	// Push it to registry/repository:$digest.sig
	signatureTag, err := name.ParseReference(fmt.Sprintf("%s/%s:%s.sig", tag.Context().RegistryStr(), tag.Context().RepositoryStr(), hex))
	if err != nil {
		return err
	}

	signatureImg, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		return err
	}
	l.Infof("Pushing signature to %s", signatureTag)
	if err := remote.Write(signatureTag, signatureImg, remote.WithAuthFromKeychain(creds)); err != nil {
		return err
	}
	return nil
}

type Critical struct {
	Identity Identity `json:"identity`
	Image    Image    `json:"image"`
	Type     string   `json:"type"`
}

type Identity struct {
	DockerReference string `json:"docker-reference"`
}

type Image struct {
	DockerManifestDigest string `json:"Docker-manifest-digest"`
}

type SimpleSigning struct {
	Critical Critical
	Optional map[string]interface{}
}

type MySignature struct {
	mediaType string
	body      []byte
}

func (s *MySignature) MediaType() (types.MediaType, error) {
	return types.DockerManifestSchema2, nil
}
func (s *MySignature) Digest() (v1.Hash, error) {
	digest, _, err := v1.SHA256(bytes.NewReader(s.body))
	return digest, err
}
func (s *MySignature) Size() (int64, error) {
	return int64(len(s.body)), nil
}
