/*
Copyright 2017 The Kubernetes Authors.

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

// This file was automatically generated by lister-gen

package v1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	api "k8s.io/client-go/pkg/api"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// PodTemplateLister helps list PodTemplates.
type PodTemplateLister interface {
	// List lists all PodTemplates in the indexer.
	List(selector labels.Selector) (ret []*v1.PodTemplate, err error)
	// PodTemplates returns an object that can list and get PodTemplates.
	PodTemplates(namespace string) PodTemplateNamespaceLister
	PodTemplateListerExpansion
}

// podTemplateLister implements the PodTemplateLister interface.
type podTemplateLister struct {
	indexer cache.Indexer
}

// NewPodTemplateLister returns a new PodTemplateLister.
func NewPodTemplateLister(indexer cache.Indexer) PodTemplateLister {
	return &podTemplateLister{indexer: indexer}
}

// List lists all PodTemplates in the indexer.
func (s *podTemplateLister) List(selector labels.Selector) (ret []*v1.PodTemplate, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.PodTemplate))
	})
	return ret, err
}

// PodTemplates returns an object that can list and get PodTemplates.
func (s *podTemplateLister) PodTemplates(namespace string) PodTemplateNamespaceLister {
	return podTemplateNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PodTemplateNamespaceLister helps list and get PodTemplates.
type PodTemplateNamespaceLister interface {
	// List lists all PodTemplates in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.PodTemplate, err error)
	// Get retrieves the PodTemplate from the indexer for a given namespace and name.
	Get(name string) (*v1.PodTemplate, error)
	PodTemplateNamespaceListerExpansion
}

// podTemplateNamespaceLister implements the PodTemplateNamespaceLister
// interface.
type podTemplateNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all PodTemplates in the indexer for a given namespace.
func (s podTemplateNamespaceLister) List(selector labels.Selector) (ret []*v1.PodTemplate, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.PodTemplate))
	})
	return ret, err
}

// Get retrieves the PodTemplate from the indexer for a given namespace and name.
func (s podTemplateNamespaceLister) Get(name string) (*v1.PodTemplate, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(api.Resource("podtemplate"), name)
	}
	return obj.(*v1.PodTemplate), nil
}
