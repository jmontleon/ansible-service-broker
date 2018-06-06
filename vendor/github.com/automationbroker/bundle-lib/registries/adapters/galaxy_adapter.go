//R
// Copyright (c) 2018 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/automationbroker/bundle-lib/bundle"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v1"
	"io/ioutil"
	"net/http"
	"strings"
)

const galaxyName = "galaxy"
const galaxySearchURL = "https://galaxy.ansible.com/api/v1/search/roles/?tags=apb"
const galaxyRoleURL = "https://galaxy.ansible.com/api/v1/roles/%v/"
const galaxyApiURL = "https://galaxy.ansible.com/api/v1"
const apbYamlURL = "https://raw.githubusercontent.com/%v/%v/%v/apb.yml"

// GalaxyAdapter - Galaxy Adapter
type GalaxyAdapter struct {
	Config Configuration
}

// GalaxyDependency - Dependency from an Ansible Galaxy role.
type GalaxyDependency struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// GalaxyRole - Role from Ansible Galaxy.
type GalaxyRole struct {
	Name         string              `json:"name"`
	Username     string              `json:"username"`
	RoleID       int                 `json:"role_id"`
	Dependencies []*GalaxyDependency `json:"dependencies"`
}

// GalaxyRoleResponse - Role Response from Ansible Galaxy.
type GalaxyRoleResponse struct {
	User   string `json:"github_user"`
	Repo   string `json:"github_repo"`
	Commit string `json:"commit"`
}

// GalaxySearchResponse - Search response for Galaxy.
type GalaxySearchResponse struct {
	Count   int           `json:"count"`
	Results []*GalaxyRole `json:"results"`
	Next    string        `json:"next"`
}

// RegistryName - Retrieve the registry name
func (r GalaxyAdapter) RegistryName() string {
	return galaxyName
}

// GetImageNames - retrieve the images
func (r GalaxyAdapter) GetImageNames() ([]string, error) {
	log.Debug("GalaxyAdapter::GetImages")
	log.Debug("BundleSpecLabel: %s", BundleSpecLabel)
	log.Debug("Loading role list with tag: [apb]")

	channel := make(chan string)
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Intial call to getNextImages this will fan out to retrieve all the values.
	imageResp, err := r.getNextImages(ctx, galaxySearchURL, channel, cancelFunc)
	// if there was an issue with the first call, return the error
	if err != nil {
		return nil, err
	}
	// If no results in the fist call then close the channel as nothing will get loaded.
	if len(imageResp.Results) == 0 {
		log.Info("canceled retrieval as no items in org")
		close(channel)
	}
	var apbData []string
	counter := 1
	for imageData := range channel {
		apbData = append(apbData, imageData)
		if counter < imageResp.Count {
			counter++
		} else {
			close(channel)
		}
	}
	// check to see if the context had an error
	if ctx.Err() != nil {
		log.Errorf("encountered an error while loading images, we may not have all the apb in the catalog - %v", ctx.Err())
		return apbData, ctx.Err()
	}

	return apbData, nil
}

// FetchSpecs - retrieve the spec for the image names.
func (r GalaxyAdapter) FetchSpecs(imageNames []string) ([]*bundle.Spec, error) {
	specs := []*bundle.Spec{}
	for _, imageName := range imageNames {
		spec, err := r.loadSpec(imageName)
		if err != nil {
			log.Errorf("Failed to retrieve spec data for image %s - %v", imageName, err)
		}
		if spec != nil {
			specs = append(specs, spec)
		}
	}
	return specs, nil
}

// getNextImages - will follow the next URL using go routines.
func (r GalaxyAdapter) getNextImages(ctx context.Context,
	url string, ch chan<- string,
	cancelFunc context.CancelFunc) (*GalaxySearchResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("unable to get next roles for url: %v - %v", url, err)
		cancelFunc()
		close(ch)
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("unable to get next roles for url: %v - %v", url, err)
		cancelFunc()
		close(ch)
		return nil, err
	}
	defer resp.Body.Close()

	imageList, err := ioutil.ReadAll(resp.Body)

	iResp := GalaxySearchResponse{}
	err = json.Unmarshal(imageList, &iResp)
	if err != nil {
		log.Errorf("unable to get next images for url: %v - %v", url, err)
		cancelFunc()
		close(ch)
		return &iResp, err
	}
	// Keep getting the images
	if iResp.Next != "" {
		log.Debugf("getting next page of results - %v", iResp.Next)
		// Fan out calls to get the next images.
		go r.getNextImages(ctx, fmt.Sprintf("%v%v", galaxyApiURL, iResp.Next), ch, cancelFunc)
	}
	for _, imageName := range iResp.Results {
		log.Debugf("Trying to load %v.%v", imageName.Username, imageName.Name)
		go func(image *GalaxyRole) {
			select {
			case <-ctx.Done():
				log.Debugf(
					"loading images failed due to context err - %v name - %v",
					ctx.Err(), image.Name)
				return
			default:
				ch <- fmt.Sprintf("%v.%v#%v", image.Username, image.Name, image.RoleID)
			}
		}(imageName)
	}
	return &iResp, nil
}

func (r GalaxyAdapter) loadSpec(imageName string) (*bundle.Spec, error) {
	spec := bundle.Spec{}
	roleId := strings.Split(imageName, "#")[1]
	req, err := http.NewRequest("GET", fmt.Sprintf(galaxyRoleURL, roleId), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	role, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	roleResp := GalaxyRoleResponse{}
	err = json.Unmarshal(role, &roleResp)
	if err != nil {
		return nil, err
	}

	specReq, err := http.NewRequest("GET", fmt.Sprintf(apbYamlURL, roleResp.User, roleResp.Repo, roleResp.Commit), nil)
	if err != nil {
		log.Errorf("unable to get apb.yaml", err)
		return nil, err
	}

	specResp, err := http.DefaultClient.Do(specReq)
	if err != nil {
		log.Errorf("unable to get apb.yaml", err)
		return nil, err
	}
	defer specResp.Body.Close()
	specFile, err := ioutil.ReadAll(resp.Body)

	err = yaml.Unmarshal(specFile, &spec)
	if err != nil {
		return nil, err
	}

	spec.Runtime = 2
	spec.Image = "ansibleplaybookbundle/apb-base"

	return &spec, nil
}
