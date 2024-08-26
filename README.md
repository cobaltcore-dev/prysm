# Prysm - Comprehensive Observability CLI Tool for Ceph and Rados Gateway Monitoring

>**Disclaimer**
>
>Prysm is currently under heavy development and may contain bugs, incomplete features, and non-functional code. This project is in the testing and proof-of-concept stage, so please use it with caution and be aware that it may not work as expected. Contributions and feedback are appreciated as we continue to improve and stabilize the tool

## Overview

Prysm is a versatile CLI tool designed to provide an efficient observability solution for a wide range of systems, including RadosGW (Rados Gateway), Ceph storage clusters, and various hardware components. With a multi-layered architecture, Prysm enables real-time monitoring, data collection, and analysis across diverse environments, ensuring optimal performance, compliance, and operational insights.

## Features

-	Multi-system Support: Prysm supports RadosGW, Ceph clusters, and hardware monitoring, making it a comprehensive observability tool.
-	Flexible Architecture: Prysm’s four-layered architecture—Consumers, NATS, Remote Producers, and Nearby Producers—enables it to handle a variety of observability tasks with precision and scalability.
-	Diverse Data Collection: Collect and analyze metrics and logs from RadosGW, Ceph, and hardware devices such as disks. Producers can be configured to gather data locally or remotely, ensuring adaptability to different environments.
-	Real-time Messaging: Use NATS as the messaging backbone to enable real-time, low-latency communication between data producers and consumers, ensuring seamless data flow.
-	Customizable Output: Prysm supports multiple output formats, including console, NATS, and Prometheus, allowing you to tailor the tool to your specific observability requirements.
-	Standalone Functionality: Prysm can be used standalone for specific tasks, such as providing a metrics endpoint for Prometheus, checking disk health, or printing data directly to the console.

## Components

### Consumers

Purpose:
  - Process and analyze data received from various systems, including RadosGW, Ceph, and hardware components.

Key Responsibilities:
  -	Generate alerts based on predefined conditions.
  -	Store and analyze logs for troubleshooting and auditing.
  -	Display real-time metrics on dashboards.
  -	Perform advanced analytics and usage reporting.
  -	Ensure regulatory compliance through log and metrics analysis.

[Monitoring Quota Usage](pkg/consumers/quotausageconsumer/README.md)  

### NATS

Purpose:
  - Acts as the messaging backbone for the system.

Key Responsibilities:
  -	Route messages between producers and consumers.
  -	Handle high volumes of messages with low latency.
  -	Ensure reliable message delivery even in the face of network issues.


### Remote Producers

Purpose:
  - Collect metrics and logs from various systems via APIs or other interfaces, typically from outside the monitored environment.  

Key Responsibilities:  
  -	Gather data using appropriate APIs or interfaces.  
  -	Transmit collected data to NATS.  
  -	Operate with minimal configuration, focusing on remote accessibility.  

[RGW Bucket Notifications](pkg/producers/bucketnotify/README.md)  
[Quota Usage Monitor](pkg/producers/quotausagemonitor/README.md)  
[RadosGW Usage Exporter](pkg/producers/radosgwusage/README.md)  


### Nearby Producers

Purpose:
- Deployed within the same network or environment as the monitored systems, allowing direct access to logs, metrics, and configuration files.

Key Responsibilities:
  -	Collect data directly from system log files, metrics endpoints, or hardware sensors (e.g., SMART attributes for disk health).
  -	Leverage proximity for lower latency and higher data fidelity.
  -	Transmit collected data to NATS.

[RGW Bucket Notifications](pkg/producers/bucketnotify/README.md)  
[Disk Health Metrics](pkg/producers/diskhealthmetrics/README.md)  
[Kernel Metrics](pkg/producers/kernelmetrics/README.md)  
[Resource Usage](pkg/producers/resourceusage/README.md)  

## Usage

Prysm can be employed across a wide range of observability scenarios, from monitoring the health of Ceph storage clusters and RadosGW instances to ensuring the reliability of hardware components through SMART attribute analysis. Whether you need to integrate with Prometheus, send real-time alerts via NATS, or simply log and visualize system performance, Prysm offers the tools and flexibility to meet your needs.

---
> This README is a draft and will be updated as Prysm continues to evolve. Contributions, suggestions, and feedback are welcome to help improve and expand the functionality of Prysm.