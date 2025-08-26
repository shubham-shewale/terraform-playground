terraform {
  backend "s3" {
    bucket = "my-terraform-state-bucket-381492134996"
    key    = "terraform-playground-basic-vpc.tfstate"
    region = "us-east-1"
  }
}
