FROM crater-harbor.act.buaa.edu.cn/docker.io/library/python:3.11-slim

ENV HTTPS_PROXY=http://192.168.5.58:1080 \
    NO_PROXY=.local,localhost,127.0.0.1

RUN pip install --no-cache-dir \
    -i https://pypi.tuna.tsinghua.edu.cn/simple \
    envd

RUN mkdir -p /workspace 
WORKDIR /workspace