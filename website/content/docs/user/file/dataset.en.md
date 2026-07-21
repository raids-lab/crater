---
title: Dataset
description: Documentation for Dataset
icon: HomeIcon
---

# Dataset

## What is a Dataset

A dataset is essentially a link that points to a specific file location, making it more convenient to mount and share. The warehouse address function currently cannot directly download from open-source communities; it mainly provides a specific description for easier sharing. If you need to download from open-source communities, you can refer to the job template for downloading models and datasets from the ModelScope community, or download them locally and upload them to the platform. For uploading large files, refer to the file system section.

## Difference between Dataset/Model and Shared Files

Datasets/models mainly provide read-only files. In the future, we will provide an option to move datasets/models to read-only shared folders and use corresponding technologies to accelerate the datasets/models, thereby improving training efficiency. Shared files mainly allow you to share personal files with others and allow others to read and write. If you need to read and write files in the public space, please contact the administrator.

## Where to View Datasets

Under `Data Management - Dataset`, you can view datasets. The datasets displayed here include those created by the user, those shared with the individual, and those shared with the account.

![dataset](./img/dataset.webp)

Each dataset has some basic description. On the right, there are four buttons: delete, personal share, account share, and rename. These operations can only be used by the dataset creator.

## How to Create a Dataset

There is a "Create Dataset" button in the top-left corner of the dataset page. Click it and then fill in the name, description, and select the folder location to create the dataset.

![dataset-create](./img/dataset-create.webp)

The name of the created dataset cannot be the same. When selecting a folder, the system will automatically show the public, personal, and current account space files that you can see, and you can then select one.

![alt text](./img/select-file.webp)

## How to Use a Dataset

On the new job page, there is a data mounting box on the right. After adding a data mount, you can select a dataset and mount it into the container.

![alt text](./img/mount.webp)