# Tekton Chains

## Installation

1. Make sure you install Tekton Pipelines first!

1. Install Chains with: `ko apply -f config/`


## Usage

To get started, you first have to generate a GPG keypair to be used by your Tekton system.
There are many ways to go about this, but you can usually use something like this:

```shell
gpg gen-key
```

Enter a passprase (make sure you remember it!) and a name for the key.

Next, you'll need to upload the private key as a Kubernetes `Secret` so Tekton can use it
to sign.
To do that, export the secret key and base64 encode it:

```shell
gpg --export-secret-key --armor $keyname | base64
```

And set that as the key `private` in the `Secret` `signing-secrets`:

```shell
kubectl edit secret signing-secrets -n tekton-pipelines
```

Do the same for your passphrase, remembering to remove any unnecessary
whitespace and base64 encode it:

```shell
echo -n 'mypassword' | base64
```

And set that as the key `passphrase` in the `Secret` `signing-secrets`:

```shell
kubectl edit secret signing-secrets -n tekton-pipelines
```

## Verification

Assuming you have the keys loaded into GPG on your system (you should if you created them earlier),
you can retrieve the signature and payload using kubectl to verify them.

They are stored in annotations on the `TaskRun`.

```shell
kubectl get taskrun $taskrun -o=json | jq -r .metadata.annotations.body | base64 --decode > body
kubectl get taskrun $taskrun -o=json | jq -r .metadata.annotations.signed > signature
```

Then verify them again with gpg:

```shell
gpg --verify signature body
```
