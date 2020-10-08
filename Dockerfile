FROM ubuntu:16.04

ARG TAG=sus
ENV env_var_name=$TAG

ARG AWS_ACCESS_KEY_ID=test
ENV access_key=$AWS_ACCESS_KEY_ID

ARG AWS_SECRET_ACCESS_KEY=test2
ENV secret_key=$AWS_SECRET_ACCESS_KEY

RUN apt-get update && apt-get install -y \
    wget \
    unzip \
  && rm -rf /var/lib/apt/lists/*
  
RUN wget --quiet https://releases.hashicorp.com/terraform/0.11.3/terraform_0.11.3_linux_amd64.zip \
  && unzip terraform_0.11.3_linux_amd64.zip \
  && mv terraform /usr/bin \
  && rm terraform_0.11.3_linux_amd64.zip

#COPY ./abc /$TAG/
RUN echo $TAG

#COPY ./abc /$env_var_name/
RUN echo $env_var_name

#COPY ./abc /$TAG/
COPY ./test.tf /$TAG/
WORKDIR /tmp/
RUN echo "hello"
RUN echo `pwd`
RUN echo `ls -las`

RUN echo $access_key

RUN echo $secret_key

RUN terraform init -var accessKey=$access_key -var secretKey=$secret_key
#RUN terraform plan -var accessKey=$access_key -var secretKey=$secret_key -out
#RUN terraform apply -var accessKey=$access_key -var secretKey=$secret_key -auto-approve
