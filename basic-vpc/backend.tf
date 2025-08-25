terraform {
  backend "s3" {
    bucket         = "my-terraform-state-bucket-590183704678"
    key            = "terraform-playground-basic-vpc.tfstate"
    region         = "us-east-1"
  }
}
