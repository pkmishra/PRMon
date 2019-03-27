

provider "aws" {
  version = "~> 1.2"
  region = ""
  profile = ""
}


resource "aws_lambda_function" "prmon" {
  function_name =  "prmon"
  handler = "main"
  runtime = "go1.x"
  filename = "../main.zip"
  source_code_hash = "${base64sha256(file("../main.zip"))}"
  role = "arn:aws:iam::<change me>"
  timeout = 10
  vpc_config {
    security_group_ids = []
    subnet_ids = []
  }
}