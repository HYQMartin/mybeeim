package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

//********产生公钥私钥文件，注：pem格式，默认放置在cfg文件夹下
func GenRsakey(bits int) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}
	derstream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derstream,
	}
	file, err := os.Create("../../cfg/private.pem")
	if err != nil {
		return err
	}

	err = pem.Encode(file, block)
	if err != nil {
		return err
	}

	// create public key
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	file, err = os.Create("../../cfg/public.pem")
	if err != nil {
		return err
	}
	err = pem.Encode(file, block)
	if err != nil {
		return err
	}
	return nil
}

/*func main() {
	err := GenRsakey(1024)
	if err != nil {
		//	fmt.Println(err)
	}
}*/

//********公钥加密 注意：os.Open 相对路径会报错，且这一块应打包在客户端
func RsaEncrypt(origData []byte, publicKey []byte) ([]byte, error) {
	/*file1, err := os.Open("D:/myproject/src/beeim/cfg/public.pem")
	if err != nil {
		panic(err)
	}
	defer file1.Close()
	publicKey, err := io util.ReadAll(file1)
	if err != nil {
		panic(err)
	}*/
	/*publicKey := []byte(`-----BEGIN PUBLIC KEY-----
	MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCbmNv4mBsuPb4xzuUwT2TxEbjG
	va76zLrF//NRiUKV/T8mKtfv/R+Q+7FGwYkZbDNl5bO0UF0MR69u8ZE1cBAzn74z
	JGqQVs9QrRgp3VTYM/9s8nOvogcz6+l1amEY+djV3iztHOxyptl1Eq8d3r82kcwf
	ybxRlNMfwNhT7VpvqQIDAQAB
	-----END PUBLIC KEY-----`)*/

	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

//********私钥解密  注意：相对路径读取还没解决，这块不能出现在客户端，否则暴露私钥
func RsaDecrypt(ciphertext []byte) ([]byte, error) {
	/*file2, err := os.Open("D:/myproject/src/beeim/cfg/private.pem")
	if err != nil {
		panic(err)
	}
	defer file2.Close()
	privateKey, err := ioutil.ReadAll(file2)
	if err != nil {
		panic(err)
	}*/
	privateKey := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQCbmNv4mBsuPb4xzuUwT2TxEbjGva76zLrF//NRiUKV/T8mKtfv
/R+Q+7FGwYkZbDNl5bO0UF0MR69u8ZE1cBAzn74zJGqQVs9QrRgp3VTYM/9s8nOv
ogcz6+l1amEY+djV3iztHOxyptl1Eq8d3r82kcwfybxRlNMfwNhT7VpvqQIDAQAB
AoGABdcrCp3LB2VR6lS1zaZtR48+vFcKZmeg6yW8YGcilLa41BnvmRaLRMnt0ZYa
K1YgZ8bDhBUwKPHX5/YxWSwnr3ll0yVKYPw310djZduJT/XSy2ugEpfhZEyKEKxE
sxArAI25IWHWnkKbJ1d6AfR0Rvac5v5O5rkpAiFWCXm633ECQQDBNWgqiSIOB4LQ
SkbMdElTOfv5/lPmjDyh2HgjRtUFs6nIDp9aJeJ4f5aZWN3trFJRTqku72dxclYi
8eSimqqjAkEAzio+ydoeF5NoYBBb5Gx5Uooz8dncxyMdzPE6OaOza9AuF5uw8CXg
ZFFXVPtbKJ7QzZ3DsenyCm4uhifpR8yNQwJAY8rsBJxUBJ8IiAD1VIDzppMafONJ
/piMcKPYWZAqUwmbNgOndu5+bPKpnIb0CeCpm+lfJSjuawA9UUtTZlEwtQJBAIT0
16NjqE56ATEau7h3gFKL0G4jm29NpVVbKLqtaPOZwW/2N0jYlHr9vj2PEL4ElhJU
sTUW88JoRla8fISSVXMCQQCGt4L8msE7Hb1yLJkIAE3SJXS6y6paFuBbensjKHEK
MlVlIJO2Pwz66P+w/Q+AGGGF6KvxIGViXM9ALZcP3ZQ8
-----END RSA PRIVATE KEY-----`)
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("private key error!")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}
