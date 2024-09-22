import os
import requests
import json
import getpass
import csv
import time

def get_env_or_input(env, prompt, is_password=False):
    value = os.environ.get(env)
    if value is None:
        if is_password:
            value = getpass.getpass(prompt)
        else:
            value = input(prompt)
    return value

def login(username, password):
    url = ***REMOVED***
    headers = {
        'accept': 'application/json',
        'Content-Type': 'application/json'
    }
    data = {
        "auth": "act",
        "password": password,
        "username": username
    }
    response = requests.post(url, headers=headers, data=json.dumps(data))
    if response.status_code == 200:
        return response.json()['data']['accessToken']
    else:
        raise Exception(f"Login failed with status code {response.status_code}")

def switch_account(token, queue):
    url = ***REMOVED***
    headers = {
        'accept': 'application/json',
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/json'
    }
    data = {
        "queue": queue
    }
    response = requests.post(url, headers=headers, data=json.dumps(data))
    if response.status_code == 200:
        return response.json()['data']['accessToken']
    else:
        raise Exception(f"Switch account failed with status code {response.status_code}")

def submit_job(token, job_name, queue, model_name, priority, batch_size, cpu, memory, gpu, duration):
    url = 'https://crater.***REMOVED***/api/v1/aijobs/training'
    headers = {
        'accept': 'application/json, text/plain, */*',
        'authorization': f'Bearer {token}',
        'content-type': 'application/json',
    }
    data = {
        "name": job_name,
        "slo": priority,
        "resource": {
            "cpu": cpu,
            "memory": memory,
            "nvidia.com/gpu": gpu
        },
        "image": "***REMOVED***/ai-portal/jupyter-tensorflow:v2.2.0",
        "command": f". /miniconda/etc/profile.d/conda.sh && conda activate base  && python run_model.py --gpus 1 --dur {duration}  --model-name {model_name} --batch-size {batch_size} --amp 0",
        "workingDir": "/dlbench",
        "volumeMounts": [
            {"subPath": "public/dnn-train-data", "mountPath": "/datasets"},
            {"subPath": "public/jupyterhub-shared/miniconda", "mountPath": "/miniconda"},
            {"subPath": "public/dlbench/dlbench", "mountPath": "/dlbench"}
        ],
        "envs": [],
        "useTensorBoard": False
    }
    response = requests.post(url, headers=headers, data=json.dumps(data))
    if response.status_code == 200:
        return True
    else:
        print(f"Submit job failed with status code {response.status_code}")
        print(response.text)
        return False

def main():
    username = get_env_or_input("EMIAS_USERNAME", "请输入用户名：")
    password = get_env_or_input("EMIAS_PASSWORD", "请输入密码：", is_password=True)
    queues = get_env_or_input("EMIAS_QUEUES", "请输入用于切换的队列，以逗号分隔：").split(',')
    token = login(username, password)
    queue_token_map = {}
    for queue in queues:
        new_token = switch_account(token, queue)
        queue_token_map[queue] = new_token
    
    csv_file_path = get_env_or_input("EMIAS_JOBS_CSV", "请输入 CSV 文件路径：")
    with open(csv_file_path, 'r') as file:
        reader = csv.reader(file)
        next(reader)  # Skip header
        for row in reader:
            job_name, queue, model_name, priority_str, batch_size_str, cpu, memory, gpu, duration_str, sleep_minutes_str = row
            priority = int(priority_str)
            batch_size = int(batch_size_str)
            cpu = int(cpu)
            gpu = int(gpu)
            duration = int(duration_str)
            job_token = queue_token_map[queue]
            sleep_time = int(sleep_minutes_str)
            # 提交作业前，先休眠一段时间
            print(f"Sleeping for {sleep_time} minutes...")
            time.sleep(sleep_time * 60)  # 将分钟转换为秒
            if submit_job(job_token, job_name, queue, model_name, priority, batch_size, cpu, memory, gpu, duration):
                print(f"Submitted job {job_name} successfully.")
            else:
                print(f"Failed to submit job {job_name}.")

if __name__ == '__main__':
    main()