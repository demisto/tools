# Docker tool
A tool to ease the creation of a Docker image with Python libs, it will create a Docker image from base image python:2.7

### How to create a new Docker image with Python libs
***Prerequisite***: These instructions assume that you have Docker installed on the machine you are running.

1. Update the requirements.txt in this folder, add the Python libs you want to this file
2. Run the script create_docker_image with first argument to be the new Docker image name, For example:  ``` ./create_docker_image.sh mycompany/image ```
  1. If you need ***sudo*** to run Docker commands, add sudo prefix to last command too, For example:         
      ``` sudo ./create_docker_image.sh mycompany/image ```
3. You should now have a new Docker image locally.

If you want to share this docker image with other machines/peers , you will need to push it to a Docker registry.
Do this on Docker hub:

1. Login to Docker Hub: ``` docker login ```
2. Push the image. For example: : ``` docker push mycompany/image ```
