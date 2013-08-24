skeleton
=======

# What is skeleton?

Shipper manages the difficult task of deploying your software to the cloud.
It's initial goals are to take your software packaged using docker and
ship it to the linode cloud. It also allows your software to manage it's
credentials in a secure manner which avoids leaving keys in source control

# ShipperFile

The ShipperFile stores a description of your deployment. It should be checked
into source control. For a detailed overview see the FileFormat documentation
or look at the examples

# Architecture Overview

There are three main components in skeleton

1. The Orchestration Server
2. The Secret Server
3. The Database Docker Containers

# The Orchestration Server

This server receives deployment information and updates the configuration to
match it. It is designed to be intelligent and only make the necessary changes

# The Secret Server

This stores all secrets and also stores deployment specific details
like server ip addresses. This can be queried using the provided command line
functions, or the built in api

# The Database Docker Containers

Everyone needs a database. There's no reason to have five different versions
of postgresql that are all compatible with the secret server. For this reason
skeleton ships with several databases. If you don't see yours, add a pull
request
