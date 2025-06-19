/*

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

package swift

const (
	RunAsUser      int64 = 42445
	ProxyPort      int32 = 8081
	ProxyHttpdPort int32 = 8080

	AccountServerPort   int32 = 6202
	ContainerServerPort int32 = 6201
	ObjectServerPort    int32 = 6200
	RsyncPort           int32 = 873

	ServiceName        = "swift"
	ServiceType        = "object-store"
	ServiceAccount     = "swift-swift"
	ServiceDescription = "Swift Object Storage"

	ClaimName = "swift"

	// Must match with settings in the dataplane-operator
	// and adoption docs/tests
	SwiftConfSecretName  = "swift-conf"
	RingConfigMapName    = "swift-ring-files"
	RingSourceSecretName = "swift-ring-source"
)
