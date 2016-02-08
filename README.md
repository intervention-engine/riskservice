Risk Service [![Build Status](https://travis-ci.org/intervention-engine/riskservice.svg?branch=master)](https://travis-ci.org/intervention-engine/riskservice)
==============================================================================================================================================================

The *riskservice* project provides a prototype risk service server for the [Intervention Engine](https://github.com/intervention-engine/ie) project. The *riskservice* server calculates risk scores for individual patients and provides risk component data to allow the Intervention Engine [frontend](https://github.com/intervention-engine/frontend) to properly draw the "risk pies". This is a proof-of-concept service only and currently supports a stroke score (based on CHA2DS2-VASc) and a negative outcome score (a simple sum of conditions and medications).

Building and Running riskservice Locally
----------------------------------------

Intervention Engine is a stack of tools and technologies. For information on installing and running the full stack, please see [Building and Running the Intervention Engine Stack in a Development Environment](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md).

For information related specifically to building and running the code in this repository (*riskservice*), please refer to the following sections in the above guide. Note that the risk service is useless without the Intervention Engine server, so it is listed as a prerequisite.

-	(Prerequisite) [Install Git](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#install-git)
-	(Prerequisite) [Install Go](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#install-go)
-	(Prerequisite) [Install MongoDB](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#install-mongodb)
-	(Prerequisite) [Run MongoDB](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#run-mongodb)
-	(Prerequisite) [Clone ie Repository](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#clone-ie-repository)
-	(Prerequisite) [Build and Run Intervention Engine Server](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#build-and-run-intervention-engine-server)
-	[Clone riskservice Repository](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#clone-riskservice-repository)
-	[Build and Run Risk Service Server](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#build-and-run-risk-service-server)
-	(Optional) [Create Intervention Engine User](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#create-intervention-engine-user)
-	(Optional) [Generate and Upload Synthetic Patient Data](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#generate-and-upload-synthetic-patient-data)

License
-------

Copyright 2016 The MITRE Corporation

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
