provider "docker" {
  host = "unix:///var/run/docker.sock"
}

resource "docker_container" "tf" {
  # Implement resoure
}

resource "docker_image" "busybox" {
  name = "inage:tag"
}

