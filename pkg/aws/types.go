/*
Copyright 2022 John Homan

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

package aws

// StatementEntry dictates what this policy allows or doesn't allow.
type StatementEntry struct {
	Effect    string                 `json:",omitempty"`
	Action    interface{}            `json:",omitempty"`
	Resource  interface{}            `json:",omitempty"`
	Principal map[string]interface{} `json:",omitempty"`
	Condition map[string]interface{} `json:",omitempty"`
}

// PolicyDocument is our definition of our policies to be uploaded to AWS Identity and Access Management (IAM).
type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}
