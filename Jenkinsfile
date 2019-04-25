node('docker') {
  timestamps {
    properties([disableConcurrentBuilds()])

    docker.withRegistry('https://quay.io', 'quay-infrajenkins-robot-creds') {
      stage('Build') {
        checkout scm

        img = docker.build("quay.io/aspenmesh/tock:${env.BRANCH_NAME}-${env.BUILD_ID}")

      }
    }
  }
}
