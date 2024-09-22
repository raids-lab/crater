import csv

bench_list = {
    'NeuMF-pre': 'benchmark_ncf',
    'bert': 'benchmark_bert',
    'resnet50': 'benchmark_imagenet',
    'mobilenet_v3_small': 'benchmark_imagenet',
    'cycle_gan': 'benchmark_img2img',
    'pix2pix': 'benchmark_img2img',
    'pointnet': 'benchmark_pointnet',
    'PPO': 'benchmark_rl',
    'dcgan': 'benchmark_dcgan',
    'ResNet18': 'benchmark_cifar',
    'MobileNetV2': 'benchmark_cifar',
    'EfficientNetB0': 'benchmark_cifar',
    'VGG': 'benchmark_cifar',
    'LSTM': 'benchmark_lstm',
    'TD3': 'benchmark_rl2',
    'transformer': 'benchmark_transformer'
}

data = [
    ['job_resnet50_1', 'q-11', 'resnet50', 0, 32, 1, '2Gi', 1, 3000, 1],
    ['job_resnet50_2', 'q-10', 'resnet50', 1, 64, 2, '4Gi', 1, 4000, 0],
    ['job_pointnet_1', 'q-10', 'pointnet', 0, 64, 1, '2Gi', 2, 1800, 0],
    ['job_pointnet_2', 'q-11', 'pointnet', 1, 128, 2, '4Gi', 1, 2400, 1],
    ['job_mobilenet_v3_small_1', 'q-12', 'mobilenet_v3_small', 1, 128, 4, '8Gi', 1, 600, 1],
    ['job_mobilenet_v3_small_2', 'q-12', 'mobilenet_v3_small', 2, 256, 5, '10Gi', 1, 1200, 2],
    ['job_NeuMF-pre_1', 'q-10', 'NeuMF-pre', 0, 32, 1, '2Gi', 1, 2000, 0],
    ['job_bert_1', 'q-10', 'bert', 0, 32, 1, '2Gi', 1, 2500, 0],
    ['job_cycle_gan_1', 'q-10', 'cycle_gan', 0, 32, 1, '2Gi', 1, 2800, 0],
    ['job_pix2pix_1', 'q-11', 'pix2pix', 0, 32, 1, '2Gi', 1, 2700, 0],
    ['job_pointnet_3', 'q-12', 'pointnet', 3, 512, 4, '10Gi', 1, 4000, 3],
    ['job_PPO_1', 'q-10', 'PPO', 0, 32, 1, '2Gi', 1, 2200, 0],
    ['job_dcgan_1', 'q-10', 'dcgan', 0, 32, 1, '2Gi', 1, 2600, 0],
    ['job_ResNet18_1', 'q-11', 'ResNet18', 0, 32, 1, '2Gi', 1, 2400, 0],
    ['job_MobileNetV2_1', 'q-10', 'MobileNetV2', 0, 32, 1, '2Gi', 1, 2300, 0],
    ['job_EfficientNetB0_1', 'q-10', 'EfficientNetB0', 0, 32, 1, '2Gi', 1, 2200, 0],
    ['job_VGG_1', 'q-10', 'VGG', 0, 32, 1, '2Gi', 1, 2500, 0],
    ['job_LSTM_1', 'q-10', 'LSTM', 0, 32, 1, '2Gi', 1, 2100, 0],
    ['job_TD3_1', 'q-11', 'TD3', 0, 32, 1, '2Gi', 1, 2300, 0],
    ['job_transformer_1', 'q-11', 'transformer', 0, 32, 1, '2Gi', 1, 2400, 0]
]

with open('output.csv', 'w', newline='') as csvfile:
    writer = csv.writer(csvfile)
    writer.writerow(['job_name', 'queue', 'model_name', 'priority', 'batch_size', 'cpu', 'memory', 'gpu', 'duration', 'sleep_minutes'])
    for row in data:
        writer.writerow(row)

for row in data:
    print(','.join(map(str, row)))