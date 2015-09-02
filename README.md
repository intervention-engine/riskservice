# riskservice
Risk assessment services intended to work with FHIR servers

Requires Go 1.5 and MongoDB 3.X

Command line flags:
* registerURL - Provide a FHIR endpoint to where the application will register a subscription
* registerENV - For use when running Dockerized. This will look at the IE_PORT_3001_TCP environment variables for a
FHIR server endpoint.

# License

Copyright 2015 The MITRE Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
