/*
Copyright 2020 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/base64"
	"flag"

	"github.com/dlorenc/chains/pkg/sign"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/apis"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/tracker"
)

var (
	namespace = flag.String("namespace", "", "Namespace to restrict informer to. Optional, defaults to all namespaces.")
)

func main() {
	flag.Parse()

	sharedmain.MainWithContext(injection.WithNamespaceScope(signals.NewContext(), *namespace), "watcher",
		func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
			logger := logging.FromContext(ctx)
			taskRunInformer := taskruninformer.Get(ctx)

			kubeclientset := kubeclient.Get(ctx)
			pipelineclientset := pipelineclient.Get(ctx)

			opt := reconciler.Options{
				KubeClientSet:     kubeclientset,
				PipelineClientSet: pipelineclientset,
				ConfigMapWatcher:  cmw,
				Logger:            logger,
			}
			signer, err := sign.NewSigner()
			if err != nil {
				logger.Fatal(err)
			}
			c := &rec{
				Base:          reconciler.NewBase(opt, "watcher", pipeline.Images{}),
				logger:        logger,
				taskRunLister: taskRunInformer.Lister(),
				signer:        signer,
			}
			impl := controller.NewImpl(c, c.logger, pipeline.TaskRunControllerName)

			taskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
				AddFunc:    impl.Enqueue,
				UpdateFunc: controller.PassNew(impl.Enqueue),
			})
			c.tracker = tracker.New(impl.EnqueueKey, controller.GetTrackerLease(ctx))

			return impl
		})
}

type rec struct {
	*reconciler.Base
	logger        *zap.SugaredLogger
	taskRunLister listers.TaskRunLister
	tracker       tracker.Interface
	signer        *sign.Signer
}

func (r *rec) Reconcile(ctx context.Context, key string) error {
	r.logger.Infof("reconciling resource key: %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Get the Task Run resource with this namespace/name
	tr, err := r.taskRunLister.TaskRuns(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The resource no longer exists, in which case we stop processing.
		r.logger.Infof("task run %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		r.logger.Errorf("Error retrieving TaskRun %q: %s", name, err)
		return err
	}

	r.logger.Infof("Sending update for %s/%s (uid %s)", namespace, name, tr.UID)

	if tr.Status.GetCondition(apis.ConditionSucceeded).IsTrue() {
		if _, ok := tr.ObjectMeta.Annotations["signed"]; !ok {
			// Sign
			r.logger.Infof("Signing %s/%s (uid %s)", namespace, name, tr.UID)
			sig, body, err := r.signer.Sign(tr)
			if err != nil {
				r.logger.Warnf("error signing %s/%s: %w", namespace, name, err)
			}
			if tr.Annotations == nil {
				tr.Annotations = map[string]string{}
			}
			tr.Annotations["signed"] = string(sig)
			tr.Annotations["body"] = base64.StdEncoding.EncodeToString(body)
			if _, err := r.PipelineClientSet.TektonV1beta1().TaskRuns(tr.Namespace).Update(tr); err != nil {
				r.logger.Warnf("Error attaching signature to %s/%s: %w", namespace, name, err)
			}

			r.logger.Infof("Signed %s/%s: %s", namespace, name, sig)

			// TODO: Also loop through resources and sign those.

			for _, rr := range tr.Status.ResourcesResult {
				name := rr.ResourceRef.Name
				for _, or := range tr.Status.TaskSpec.Resources.Outputs {
					if or.Name == name {
						// Now we have the actual OR definition
						switch or.Type {
						case "image":
							r.logger.Infof("Found image resource to sign %s", or.Name)
							if err := sign.AttachSignature(or, r.signer, tr, r.logger); err != nil {
								r.logger.Warnf("Error attaching signature: %s", err)
							}
						default:
							r.logger.Infof("Can't sign resource %s of type %s", or.Name, or.Type)
						}
					}
				}

			}
		}
	}

	return nil
}
