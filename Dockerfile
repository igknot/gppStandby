FROM golang
RUN mkdir oreclient_install_dir
RUN apt-get update
RUN apt-get install libaio1 libaio-dev
RUN apt-get install unzip -y

WORKDIR /oreclient_install_dir/
RUN curl -o instantclient-basic-linux.x64-12.2.0.1.0.zip http://plinrepo1v.standardbank.co.za/repo/software/oracle/instant-client-12/instantclient-basic-linux.x64-12.2.0.1.0.zip
RUN curl -o instantclient-sdk-linux.x64-12.2.0.1.0.zip http://plinrepo1v.standardbank.co.za/repo/software/oracle/instant-client-12/instantclient-sdk-linux.x64-12.2.0.1.0.zip

RUN cd /oreclient_install_dir ; unzip /oreclient_install_dir/instantclient-basic-linux.x64-12.2.0.1.0.zip
RUN cd /oreclient_install_dir ; unzip /oreclient_install_dir/instantclient-sdk-linux.x64-12.2.0.1.0.zip

ENV PKG_CONFIG_PATH "/oreclient_install_dir/instantclient_12_2"
ENV LD_LIBRARY_PATH "/oreclient_install_dir/instantclient_12_2"

RUN ln -s /oreclient_install_dir/instantclient_12_2/libclntsh.so.12.1 /usr/lib/libclntsh.dylib
RUN ln -s /oreclient_install_dir/instantclient_12_2/libclntsh.so.12.1 /usr/lib/libclntsh.so
RUN ln -s /oreclient_install_dir/instantclient_12_2/libocci.so.12.1 /usr/lib/libocci.dylib
RUN ln -s /oreclient_install_dir/instantclient_12_2/libocci.so.12.1 /usr/lib/libocci.so

WORKDIR /go/src/github.com/igknot/
RUN git -c http.sslVerify=false clone -v https://gitlab.standardbank.co.za/a149651/gppreport.git

WORKDIR /go/src/github.com/igknot/gppStandby

RUN go get -d -v -insecure  ./...

RUN cp database/clientSoftware/oci8_linux.pc /oreclient_install_dir/instantclient_12_2/oci8.pc

RUN go install -v ./...
#----------------------------
#----------------------------
#----------------------------
FROM bitnami/minideb

RUN apt-get update
RUN apt-get install libaio1 libaio-dev curl  unzip -y

RUN  mkdir -p /go/bin/
WORKDIR /oreclient_install_dir/

RUN curl -o instantclient-basic-linux.x64-12.2.0.1.0.zip http://plinrepo1v.standardbank.co.za/repo/software/oracle/instant-client-12/instantclient-basic-linux.x64-12.2.0.1.0.zip
RUN curl -o instantclient-sdk-linux.x64-12.2.0.1.0.zip http://plinrepo1v.standardbank.co.za/repo/software/oracle/instant-client-12/instantclient-sdk-linux.x64-12.2.0.1.0.zip

RUN cd /oreclient_install_dir ; unzip /oreclient_install_dir/instantclient-basic-linux.x64-12.2.0.1.0.zip
RUN cd /oreclient_install_dir ; unzip /oreclient_install_dir/instantclient-sdk-linux.x64-12.2.0.1.0.zip

RUN ln -s /oreclient_install_dir/instantclient_12_2/libclntsh.so.12.1 /usr/lib/libclntsh.dylib
RUN ln -s /oreclient_install_dir/instantclient_12_2/libclntsh.so.12.1 /usr/lib/libclntsh.so
RUN ln -s /oreclient_install_dir/instantclient_12_2/libocci.so.12.1 /usr/lib/libocci.dylib
RUN ln -s /oreclient_install_dir/instantclient_12_2/libocci.so.12.1 /usr/lib/libocci.so

COPY --from=0 /go/src/github.com/igknot/gppStandby/database/clientSoftware/oci8_linux.pc /oreclient_install_dir/instantclient_12_2/oci8.pc
WORKDIR /go/bin/
COPY --from=0 /go/bin/ .


ENV PKG_CONFIG_PATH "/oreclient_install_dir/instantclient_12_2"
ENV LD_LIBRARY_PATH "/oreclient_install_dir/instantclient_12_2"
RUN rm -f /oreclient_install_dir/instant*.zip
RUN rm -fr /var/lib/apt/lists
ENTRYPOINT /go/bin/gppreport



