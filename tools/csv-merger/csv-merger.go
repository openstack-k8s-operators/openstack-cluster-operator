/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2020 Red Hat, Inc.
 *
 */

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
	"github.com/openstack-k8s-operators/lib-common/pkg/util"
	"github.com/openstack-k8s-operators/openstack-cluster-operator/pkg/operator"

	csvv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const operatorName = "openstack-cluster-operator"

const csvMode = "CSV"

var validOutputModes = []string{csvMode}

// TODO: get rid of this once RelatedImages officially
// appears in github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators
type relatedImage struct {
	Name string `json:"name"`
	Ref  string `json:"image"`
}

type clusterServiceVersionSpecExtended struct {
	csvv1alpha1.ClusterServiceVersionSpec
	RelatedImages []relatedImage `json:"relatedImages,omitempty"`
}

type clusterServiceVersionExtended struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   clusterServiceVersionSpecExtended       `json:"spec"`
	Status csvv1alpha1.ClusterServiceVersionStatus `json:"status"`
}

var (
	outputMode          = flag.String("output-mode", csvMode, "Working mode: "+strings.Join(validOutputModes, "|"))
	novaCsv             = flag.String("nova-csv", "", "Nova CSV string")
	neutronCsv          = flag.String("neutron-csv", "", "Neutron CSV string")
	computeNodeCsv      = flag.String("compute-node-csv", "", "Compute Node CSV string")
	keystoneCsv         = flag.String("keystone-csv", "", "Keystone CSV string")
	heatCsv             = flag.String("heat-csv", "", "Heat CSV string")
	glanceCsv           = flag.String("glance-csv", "", "Glance CSV string")
	mariadbCsv          = flag.String("mariadb-csv", "", "Mariadb CSV string")
	placementCsv        = flag.String("placement-csv", "", "Placement CSV string")
	ovnCsv              = flag.String("ovn-csv", "", "OVN CSV string")
	operatorImage       = flag.String("operator-image-name", "", "OpenStack Cluster Operator image")
	csvVersion          = flag.String("csv-version", "", "CSV version")
	replacesCsvVersion  = flag.String("replaces-csv-version", "", "CSV version to replace")
	metadataDescription = flag.String("metadata-description", "", "Metadata")
	specDescription     = flag.String("spec-description", "", "Description")
	specDisplayName     = flag.String("spec-displayname", "", "Display Name")
	namespace           = flag.String("namespace", "openstack", "Namespace")
	crdDisplay          = flag.String("crd-display", "OpenStack Cluster", "Label show in OLM UI about the primary CRD")
	csvOverrides        = flag.String("csv-overrides", "", "CSV like string with punctual changes that will be recursively applied (if possible)")
	visibleCRDList      = flag.String("visible-crds-list", "controlplanes.controlplane.openstack.org,computenodeopenstacks.compute-node.openstack.org",
		"Comma separated list of all the CRDs that should be visible in OLM console")
	relatedImagesList = flag.String("related-images-list", "",
		"Comma separated list of all the images referred in the CSV (just the image pull URLs or eventually a set of 'image|name' collations)")
	crdDir = flag.String("crds-dir", "", "the directory containing the CRDs for apigroup validation. The validation will be performed if and only if the value is non-empty.")
)

func ioReadDir(root string) ([]string, error) {
	var files []string
	fileInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, filepath.Join(root, file.Name()))
	}
	return files, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func validateNoAPIOverlap(crdDir string) bool {
	var (
		crdFiles []string
		err      error
	)
	crdFiles, err = ioReadDir(crdDir)
	if err != nil {
		panic(err)
	}

	// crdMap is populated with operator names as keys and a slice of associated api groups as values.
	crdMap := make(map[string][]string)

	for _, crdFilePath := range crdFiles {
		file, err := os.Open(crdFilePath)
		if err != nil {
			panic(err)
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		err = file.Close()
		if err != nil {
			panic(err)
		}

		crdFileName := filepath.Base(crdFilePath)
		reg := regexp.MustCompile(`([^\d]+)`)
		operator := reg.FindString(crdFileName)

		var crd apiextensions.CustomResourceDefinition
		err = yaml.Unmarshal(content, &crd)
		if err != nil {
			panic(err)
		}
		if !stringInSlice(crd.Spec.Group, crdMap[operator]) {
			crdMap[operator] = append(crdMap[operator], crd.Spec.Group)
		}
	}

	// overlapsMap is populated with collisions found - API Groups as keys,
	// and slice containing operators using them, as values.
	overlapsMap := make(map[string][]string)
	for operator := range crdMap {
		for _, apigroup := range crdMap[operator] {
			for comparedOperator := range crdMap {
				if operator == comparedOperator {
					continue
				}
				if stringInSlice(apigroup, crdMap[comparedOperator]) {
					overlappingOperators := []string{operator, comparedOperator}
					for _, o := range overlappingOperators {
						if !stringInSlice(o, overlapsMap[apigroup]) {
							overlapsMap[apigroup] = append(overlapsMap[apigroup], o)
						}
					}
				}
			}
		}
	}

	// if at least one overlap found - emit an error.
	if len(overlapsMap) != 0 {
		log.Print("ERROR: Overlapping API Groups were found between different operators.")
		for apigroup := range overlapsMap {
			fmt.Print("The API Group " + apigroup + " is being used by these operators: " + strings.Join(overlapsMap[apigroup], ", ") + "\n")
			return false
		}
	}
	return true
}

func main() {
	flag.Parse()

	if *crdDir != "" {
		result := validateNoAPIOverlap(*crdDir)
		if result {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	switch *outputMode {
	case csvMode:
		if *specDisplayName == "" || *specDescription == "" {
			panic(errors.New("Must specify spec-displayname and spec-description"))
		}

		csvs := []string{
			*novaCsv,
			*neutronCsv,
			*computeNodeCsv,
			*keystoneCsv,
			*heatCsv,
			*glanceCsv,
			*mariadbCsv,
			*placementCsv,
			*ovnCsv,
		}

		version := semver.MustParse(*csvVersion)
		var replaces string
		if *replacesCsvVersion != "" {
			replaces = fmt.Sprintf("%v.v%v", operatorName, semver.MustParse(*replacesCsvVersion).String())
		}

		// This is the basic CSV without an InstallStrategy defined
		csvBase := operator.GetCSVBase(
			operatorName,
			*namespace,
			*specDisplayName,
			*specDescription,
			*operatorImage,
			replaces,
			version,
			*crdDisplay,
		)
		csvExtended := clusterServiceVersionExtended{
			TypeMeta:   csvBase.TypeMeta,
			ObjectMeta: csvBase.ObjectMeta,
			Spec:       clusterServiceVersionSpecExtended{ClusterServiceVersionSpec: csvBase.Spec},
			Status:     csvBase.Status}

		// This is the base deployment + rbac for the OpenStack Cluster CSV
		installStrategyBase := operator.GetInstallStrategyBase(
			*namespace,
			*operatorImage,
			"IfNotPresent",
		)

		for _, image := range strings.Split(*relatedImagesList, ",") {
			if image != "" {
				name := ""
				if strings.Contains(image, "|") {
					imageSplit := strings.Split(image, "|")
					image = imageSplit[0]
					name = imageSplit[1]
				} else {
					names := strings.Split(strings.Split(image, "@")[0], "/")
					name = names[len(names)-1]
				}
				csvExtended.Spec.RelatedImages = append(
					csvExtended.Spec.RelatedImages,
					relatedImage{
						Name: name,
						Ref:  image,
					})
			}
		}

		for _, csvStr := range csvs {
			if csvStr != "" {
				csvBytes := []byte(csvStr)

				csvStruct := &csvv1alpha1.ClusterServiceVersion{}

				err := yaml.Unmarshal(csvBytes, csvStruct)
				if err != nil {
					panic(err)
				}

				installStrategyBase.DeploymentSpecs = append(installStrategyBase.DeploymentSpecs, csvStruct.Spec.InstallStrategy.StrategySpec.DeploymentSpecs...)
				installStrategyBase.ClusterPermissions = append(installStrategyBase.ClusterPermissions, csvStruct.Spec.InstallStrategy.StrategySpec.ClusterPermissions...)
				installStrategyBase.Permissions = append(installStrategyBase.Permissions, csvStruct.Spec.InstallStrategy.StrategySpec.Permissions...)

				for _, owned := range csvStruct.Spec.CustomResourceDefinitions.Owned {
					csvExtended.Spec.CustomResourceDefinitions.Owned = append(
						csvExtended.Spec.CustomResourceDefinitions.Owned,
						csvv1alpha1.CRDDescription{
							Name:        owned.Name,
							Version:     owned.Version,
							Kind:        owned.Kind,
							Description: owned.Description,
							DisplayName: owned.DisplayName,
						},
					)
				}

				csvBaseAlmString := csvExtended.Annotations["alm-examples"]
				csvStructAlmString := csvStruct.Annotations["alm-examples"]
				var baseAlmCrs []interface{}
				var structAlmCrs []interface{}
				if err = json.Unmarshal([]byte(csvBaseAlmString), &baseAlmCrs); err != nil {
					panic(err)
				}
				if err = json.Unmarshal([]byte(csvStructAlmString), &structAlmCrs); err == nil {
					//panic(err)
					for _, cr := range structAlmCrs {
						baseAlmCrs = append(
							baseAlmCrs,
							cr,
						)
					}
				}
				almB, err := json.Marshal(baseAlmCrs)
				if err != nil {
					panic(err)
				}
				csvExtended.Annotations["alm-examples"] = string(almB)

			}
		}

		hiddenCrds := []string{}
		visibleCrds := strings.Split(*visibleCRDList, ",")
		for _, owned := range csvExtended.Spec.CustomResourceDefinitions.Owned {
			found := false
			for _, name := range visibleCrds {
				if owned.Name == name {
					found = true
				}
			}
			if !found {
				hiddenCrds = append(
					hiddenCrds,
					owned.Name,
				)
			}
		}

		csvExtended.Spec.CustomResourceDefinitions.Required = append(
			csvExtended.Spec.CustomResourceDefinitions.Required,
			csvv1alpha1.CRDDescription{
				Name:        "interconnects.interconnectedcloud.github.io",
				Version:     "v1alpha1",
				Kind:        "Interconnect",
				Description: "AMQ Interconnect",
				DisplayName: "An instance of AMQ Interconnect",
			},
		)

		hiddenCrdsJ, err := json.Marshal(hiddenCrds)
		if err != nil {
			panic(err)
		}
		csvExtended.Annotations["operators.operatorframework.io/internal-objects"] = string(hiddenCrdsJ)

		csvExtended.Spec.InstallStrategy.StrategyName = "deployment"
		csvExtended.Spec.InstallStrategy = csvv1alpha1.NamedInstallStrategy{
			StrategyName: "deployment",
			StrategySpec: installStrategyBase,
		}

		if *metadataDescription != "" {
			csvExtended.Annotations["description"] = *metadataDescription
		}
		if *specDescription != "" {
			csvExtended.Spec.Description = *specDescription
		}
		if *specDisplayName != "" {
			csvExtended.Spec.DisplayName = *specDisplayName
		}

		if *csvOverrides != "" {
			csvOBytes := []byte(*csvOverrides)

			csvO := &clusterServiceVersionExtended{}

			err := yaml.Unmarshal(csvOBytes, csvO)
			if err != nil {
				panic(err)
			}

			err = mergo.Merge(&csvExtended, csvO, mergo.WithOverride)
			if err != nil {
				panic(err)
			}

		}

		util.MarshallObject(csvExtended, os.Stdout)

	default:
		panic("Unsupported output mode: " + *outputMode)
	}

}
