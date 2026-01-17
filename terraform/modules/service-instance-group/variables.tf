variable "service_name" {
  description = "Name of the service (e.g., 'tasker')"
  type        = string
}

variable "container_registry_id" {
  description = "ID of the Yandex Container Registry"
  type        = string
}

variable "docker_compose_file" {
  description = "Path to docker-compose file for the service"
  type        = string
}

variable "domain" {
  description = "Base domain name"
  type        = string
}

variable "dns_zone_id" {
  description = "ID of the DNS zone"
  type        = string
}

variable "http_router_id" {
  description = "ID of the ALB HTTP router"
  type        = string
}

variable "alb_ip_address" {
  description = "IP address of the Application Load Balancer"
  type        = string
}

variable "network_id" {
  description = "ID of the VPC network"
  type        = string
}

variable "folder_id" {
  description = "ID of the Yandex Cloud folder"
  type        = string
}

variable "subnet_id" {
  description = "ID of the subnet for instances"
  type        = string
}

variable "alb_security_group_id" {
  description = "ID of the ALB security group (for ingress rules)"
  type        = string
}

variable "bastion_security_group_id" {
  description = "ID of the bastion security group (for SSH access)"
  type        = string
}

variable "instance_count" {
  description = "Number of instances in the group"
  type        = number
  default     = 2
}

variable "instance_memory" {
  description = "Memory in GB for each instance"
  type        = number
  default     = 1
}

variable "instance_cores" {
  description = "Number of CPU cores for each instance"
  type        = number
  default     = 2
}

variable "instance_core_fraction" {
  description = "CPU core fraction (percentage)"
  type        = number
  default     = 20
}

variable "instance_disk_size" {
  description = "Boot disk size in GB"
  type        = number
  default     = 15
}

variable "instance_zone" {
  description = "Availability zone for instances"
  type        = string
  default     = "ru-central1-d"
}

variable "http_port" {
  description = "HTTP port for the service"
  type        = number
  default     = 80
}

variable "grpc_port" {
  description = "GRPC port for the service"
  type        = number
  default     = 5051
}

variable "liveliness_check" {
  type = object({
    healthy_threshold   = number
    unhealthy_threshold = number
    interval            = number
    timeout             = number
    http_options = object({
      path = string
      port = number
    })
  })
  default = {
    healthy_threshold   = 2
    unhealthy_threshold = 3
    interval            = 2
    timeout             = 1
    http_options = {
      path = "/liveliness"
      port = 80
    }
  }
}

variable "health_check" {
  type = object({
    interval            = string
    timeout             = string
    healthcheck_port = number
    http_healthcheck = object({
      path = string
    })
  })
  default = {
    interval            = "2s"
    timeout             = "1s"
    healthcheck_port = 80
    http_healthcheck = {
      path = "/health"
    }
  }
}

variable "backend_timeout" {
  description = "Backend timeout in seconds"
  type        = string
  default     = "60s"
}

variable "preemptible" {
  description = "Use preemptible instances"
  type        = bool
  default     = true
}


