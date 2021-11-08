package firefly

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/wumansgy/goEncrypt"
	"testing"
)

const (
	private_Key = `-----BEGIN  WUMAN RSA PRIVATE KEY -----
MIIEogIBAAKCAQEAzifwFhO6X6ZSkNsj5Oh4+Ls/oQLw+f6S2t0mQz7mb0oHmF/T
USaFAc8Kv9hugbnh0uNMeODhW9RrvSLO31Xv4yUBDLUU0+c7l2br9PwnPe1fyuEC
SyCDN7mKwsgwtLureyeGVq7LVk8tLucO+WdcIxBqEr3V+YAzqd6/vRa518WFFBGm
xdnLq64bQoUE7vHozz4kzVIqqvze7xnix1kLmzobl7pW+PRs2WkkR5qXyvdMudmW
+DhfcaR+DMRSUpRSsd1kvVwidbndYoaIviwI9Z1DS6Gy1JQsvm56zuxuL7hYfU1F
/YH3tRD3gFadKyp34SX7l67IqBLQC1ZSxt5btwIDAQABAoIBAE66qdGEjzRgDEAN
sFOHBEvZFp+iw+x08BhtRGOB7faqSuRCFC11jG34Z56ERQ99sWQcMeQn3Wh4YvzE
AkUASLByPUhhDycts3BKeeoBMetQb7jG3V1beUEatodUdGVRFlfd7EvGTRkU+6hh
zTIp6uHpLtkhbknuT8ybqJXJzAc+U+Zws+flIFqcn6sh89F+/qPXiJyFZCLEjp2t
ON3Qn2+GHQP8LwkbOJlLqViQA3W6U8L5t0cwpWpzM+edInWTmaX+M39H3ys724+A
jcv6uprlyatIkNhOLEJI+1wgozgleUtYoPnIaUUxSb5mVh93UJT4qQ4fAHsq16jB
qVylJMECgYEA4Ny8PkLHHvukIq2U4UYue5EPN2skQCMWzc8dz2s+xNwvPf+UAmi/
COcHvlpgPV7mECQsgdunbWixwQLhgUbXVdX4OBBL1KTv5b0MOqXU8ITb+SrFmpCb
7sR+MOlKi4FxCY7BcsH9gxdxg1lbAHFv7LNG2SVDOyZrFTkhGvuV+GkCgYEA6rQT
JOppXoDa2wDaYjj/fMLi/GYBOyvLNonfSXVKYO4JeBCWu0keF8Jiri3Gz7SMiWh9
4qe3URZ8BbbcSj8v7G+jSZZSZW6lUhLCIxa9O1LXvuLvCWedsXjsbQfrIOc5V4Px
8ZDumLn9uxnESng128b1ss38R860GQzydxsKLx8CgYA7cl/Z3fGigUh9WoKXo+Q6
CrmJHywwQJaQxobNBT9M8CEVNPI+SM7oXZuweVgkIWiVL9sMYbO7uwfzTP2tHbtk
F/NNbxF9IDXD+Ny4zIqlI5q8HtCq8jwnPY9XAvYQN6JYsoL2Ac8xzwrVfNQQI+1B
GIxMcAt8IcYBkF7uMUEOsQKBgDs4JHx0CRInQRFxLakK6Kv6IHu+4SCk9ClWsFhA
l/vNE+aPiPjIgidMjMmWE0vlKnChROIjn0V+ftySPxMczmLB6Flw7GlbeaszwHhK
DIUjafxoFhgxZMCa2kzIarNqpDVIvYtOHmW6yCKlZbnEixJhKS1se/NCXH7VnXgg
AnnRAoGAOzUTBtTkc0d7N3e9DO0Y+1Qdbi45N1XuM9X7Xq28rKzkRiEl7MGzDQfX
U+on3+Y0YlfBJNkzN5F+KdYfILCG2WqXME6PA5tkZtHo/IkuU8NLiKRTKO4u/Y2Q
e1gYtrLF8wdEqomjjbTpgKnE6DBs8arhNtVyPC82Jope5bWh8gQ=
-----END  WUMAN RSA PRIVATE KEY -----`
	public_Key = `-----BEGIN  WUMAN  RSA PUBLIC KEY -----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzifwFhO6X6ZSkNsj5Oh4
+Ls/oQLw+f6S2t0mQz7mb0oHmF/TUSaFAc8Kv9hugbnh0uNMeODhW9RrvSLO31Xv
4yUBDLUU0+c7l2br9PwnPe1fyuECSyCDN7mKwsgwtLureyeGVq7LVk8tLucO+Wdc
IxBqEr3V+YAzqd6/vRa518WFFBGmxdnLq64bQoUE7vHozz4kzVIqqvze7xnix1kL
mzobl7pW+PRs2WkkR5qXyvdMudmW+DhfcaR+DMRSUpRSsd1kvVwidbndYoaIviwI
9Z1DS6Gy1JQsvm56zuxuL7hYfU1F/YH3tRD3gFadKyp34SX7l67IqBLQC1ZSxt5b
twIDAQAB
-----END  WUMAN  RSA PUBLIC KEY -----`
)

func Test_Rsa(t *testing.T) {
	goEncrypt.GetRsaKey()
}

func Test_encoder(t *testing.T) {
	plaintext := []byte("123456")
	pkey := []byte(public_Key)
	crypttext, err := goEncrypt.RsaEncrypt(plaintext, pkey)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("密文:", hex.EncodeToString(crypttext))

}

func Test_Decoder(t *testing.T) {
	str := "73db120e36e5f7bdc20fb8aac1138b89c5937747c228dd6eec7067fcd820bd3da328fb8930a01efe137602e657d6331a0f043a8ddc2a067b6813ad3ece34cd0a3f56a5b78af472150aae2c0a82236915b6f2a4680a758587830586dbbe34f79644a9f6bb832b53b87201f5463592072acdac1f581f19384785dd4b964af3beb8c9f83bd7be65f04c2f977b34e00c24ec57ffa730cf906917c654063307e4947fea6471c8c6ab79d096c3252a514987ace6e1ac3e86ea63a09f7fc28ba4255cb3295f6e222d44a5f82e4f9ea234898cfa17a33e79b68fab4765bc3abfa111e3aa9a44dc50efaf5b32bd1f74b99501603bda742732a0fce9437c1329e13660dafd"

	crypttext, _ := hex.DecodeString(str)
	// 解密操作，直接传入密文和私钥解密操作，得到明文
	plaintext, err := goEncrypt.RsaDecrypt(crypttext, []byte(private_Key))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("明文：", string(plaintext))

}

func Test_Decode_Base64(t *testing.T) {
	str := "sHiFX5raoRYeUgkgACaWQHWUymtDlgBVOMeRsgSCoSH3rnoGSHNSsOhE9RPqOLgXmrkC3jl/yUW/yhescXfeqgW6J9QqTV+M86cU+hieh2hky380amgDbS1AfjRvtrqL82E+erG4Qo0Om2kzg7HyKToZRk8ov2kvl214HLp9tcCv1uZlOIhmOc5YxUXLZukIy5g7Iahr6AU1V8vChmm9Czne62O3Bzh6nakDiFdeGuw0tBVombRNAKsLBxS4WiZFkuNzFjMTfKsGF9v0BDKXMuk0DcsZHMI+Oinrno2C4n0YvkhMKuM96KELsN+EczuBOKTbPuKprjBepkGtMbkxxg=="
	crypttext, _ := base64.StdEncoding.DecodeString(str)

	// 解密操作，直接传入密文和私钥解密操作，得到明文
	plaintext, err := goEncrypt.RsaDecrypt(crypttext, []byte(private_Key))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("明文：", string(plaintext))

}

func Test_Encode_Base64(t *testing.T) {
	plaintext := []byte("123456")
	pkey := []byte(public_Key)
	crypttext, err := goEncrypt.RsaEncrypt(plaintext, pkey)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("密文:", base64.StdEncoding.EncodeToString(crypttext))
}

func Test_Decode_Base64_RawURLEncoding(t *testing.T) {
	str := "sHiFX5raoRYeUgkgACaWQHWUymtDlgBVOMeRsgSCoSH3rnoGSHNSsOhE9RPqOLgXmrkC3jl/yUW/yhescXfeqgW6J9QqTV M86cU hieh2hky380amgDbS1AfjRvtrqL82E erG4Qo0Om2kzg7HyKToZRk8ov2kvl214HLp9tcCv1uZlOIhmOc5YxUXLZukIy5g7Iahr6AU1V8vChmm9Czne62O3Bzh6nakDiFdeGuw0tBVombRNAKsLBxS4WiZFkuNzFjMTfKsGF9v0BDKXMuk0DcsZHMI Oinrno2C4n0YvkhMKuM96KELsN EczuBOKTbPuKprjBepkGtMbkxxg=="
	crypttext, _ := base64.RawURLEncoding.DecodeString(str)

	// 解密操作，直接传入密文和私钥解密操作，得到明文
	plaintext, err := goEncrypt.RsaDecrypt(crypttext, []byte(private_Key))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("明文：", string(plaintext))

}
