// Copyright 2019 Omnition Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sprocessor/observability"
	gocache "github.com/patrickmn/go-cache"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	
	"time"
)

// fakeOwnerCache is a simple structure which aids querying for owners
type fakeOwnerCache struct {
	objectOwnersCache *gocache.Cache
	apiCallDuration   time.Duration
}

// NewOwnerProvider creates new instance of the owners api
func newFakeOwnerProvider(clientset *kubernetes.Clientset) OwnerAPI {
	ownerCache := fakeOwnerCache{}
	ownerCache.objectOwnersCache = gocache.New(15*time.Minute, 30*time.Minute)
	ownerCache.apiCallDuration = 50 * time.Millisecond
	return &ownerCache
}

// GetNamespace retrieves relevant metadata from API or from cache
func (op *fakeOwnerCache) GetNamespace(namespace string) *ObjectOwner {
	oo := ObjectOwner{
		UID:       "33333-66666",
		namespace: namespace,
		ownerUIDs: []types.UID{},
		kind:      "namespace",
		name:      namespace,
	}

	return &oo
}

func (op *fakeOwnerCache) deepCacheObject(namespace string, kind string, name string, objectUID types.UID) {
	time.Sleep(op.apiCallDuration)

	observability.RecordAPICallMade()

	oo := ObjectOwner{
		UID:       objectUID,
		namespace: namespace,
		ownerUIDs: []types.UID{},
		kind:      kind,
		name:      name,
	}

	op.objectOwnersCache.Add(string(oo.UID), &oo, gocache.DefaultExpiration)
}

// GetOwners fetches deep tree of owners for a given pod
func (op *fakeOwnerCache) GetOwners(pod *api_v1.Pod) []*ObjectOwner {
	objectOwners := []*ObjectOwner{}

	// Make sure the tree is cached/traversed first
	for _, or := range pod.OwnerReferences {
		_, found := op.objectOwnersCache.Get(string(or.UID))
		if !found {
			op.deepCacheObject(pod.Namespace, or.Kind, or.Name, or.UID)
		}
	}
	oo := ObjectOwner{
		UID:       "12345",
		namespace: pod.Namespace,
		ownerUIDs: []types.UID{},
		kind:      "ReplicaSet",
		name:      "SomeReplicaSet",
	}

	objectOwners = append(objectOwners, &oo)
	return objectOwners
}
