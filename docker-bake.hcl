variable "IMAGE_TAG" {
  default = "dev"
}

variable "IMAGE_REGISTRY" {
  default = "my-registry.local:5000"
}

function "gentags" {
  params = [image_name]
  result = ["${IMAGE_REGISTRY}/${image_name}:${IMAGE_TAG}"]
}

group "default" {
  targets = ["manager", "broadcast"]
}

target "manager" {
  context = "."
  dockerfile = "Dockerfile"
  tags = gentags("manager")
}

target "broadcast" {
  context = "."
  dockerfile = "broadcast/Dockerfile"
  tags = gentags("broadcast")
}
