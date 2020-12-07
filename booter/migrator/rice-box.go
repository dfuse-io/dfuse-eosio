// Code generated by rice embed-go; DO NOT EDIT.
package migrator

import (
	"time"

	"github.com/GeertJohan/go.rice/embedded"
)

func init() {

	// define files
	file2 := &embedded.EmbeddedFile{
		Filename:    "migrator.abi",
		FileModTime: time.Unix(1603825624, 0),

		Content: string("{\n    \"____comment\": \"This file was generated with eosio-abigen. DO NOT EDIT \",\n    \"version\": \"eosio::abi/1.1\",\n    \"types\": [],\n    \"structs\": [\n        {\n            \"name\": \"eject\",\n            \"base\": \"\",\n            \"fields\": [\n                {\n                    \"name\": \"account\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"table\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"scope\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"id\",\n                    \"type\": \"name\"\n                }\n            ]\n        },\n        {\n            \"name\": \"idxc\",\n            \"base\": \"\",\n            \"fields\": [\n                {\n                    \"name\": \"table\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"scope\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"payer\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"id\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"secondary\",\n                    \"type\": \"checksum256\"\n                }\n            ]\n        },\n        {\n            \"name\": \"idxdbl\",\n            \"base\": \"\",\n            \"fields\": [\n                {\n                    \"name\": \"table\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"scope\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"payer\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"id\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"secondary\",\n                    \"type\": \"float64\"\n                }\n            ]\n        },\n        {\n            \"name\": \"idxi\",\n            \"base\": \"\",\n            \"fields\": [\n                {\n                    \"name\": \"table\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"scope\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"payer\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"id\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"secondary\",\n                    \"type\": \"uint64\"\n                }\n            ]\n        },\n        {\n            \"name\": \"idxii\",\n            \"base\": \"\",\n            \"fields\": [\n                {\n                    \"name\": \"table\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"scope\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"payer\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"id\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"secondary\",\n                    \"type\": \"uint128\"\n                }\n            ]\n        },\n        {\n            \"name\": \"idxldbl\",\n            \"base\": \"\",\n            \"fields\": [\n                {\n                    \"name\": \"table\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"scope\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"payer\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"id\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"secondary\",\n                    \"type\": \"float128\"\n                }\n            ]\n        },\n        {\n            \"name\": \"inject\",\n            \"base\": \"\",\n            \"fields\": [\n                {\n                    \"name\": \"table\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"scope\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"payer\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"id\",\n                    \"type\": \"name\"\n                },\n                {\n                    \"name\": \"data\",\n                    \"type\": \"bytes\"\n                }\n            ]\n        }\n    ],\n    \"actions\": [\n        {\n            \"name\": \"eject\",\n            \"type\": \"eject\",\n            \"ricardian_contract\": \"\"\n        },\n        {\n            \"name\": \"idxc\",\n            \"type\": \"idxc\",\n            \"ricardian_contract\": \"\"\n        },\n        {\n            \"name\": \"idxdbl\",\n            \"type\": \"idxdbl\",\n            \"ricardian_contract\": \"\"\n        },\n        {\n            \"name\": \"idxi\",\n            \"type\": \"idxi\",\n            \"ricardian_contract\": \"\"\n        },\n        {\n            \"name\": \"idxii\",\n            \"type\": \"idxii\",\n            \"ricardian_contract\": \"\"\n        },\n        {\n            \"name\": \"idxldbl\",\n            \"type\": \"idxldbl\",\n            \"ricardian_contract\": \"\"\n        },\n        {\n            \"name\": \"inject\",\n            \"type\": \"inject\",\n            \"ricardian_contract\": \"\"\n        }\n    ],\n    \"tables\": [],\n    \"ricardian_clauses\": [],\n    \"variants\": []\n}"),
	}
	file3 := &embedded.EmbeddedFile{
		Filename:    "migrator.wasm",
		FileModTime: time.Unix(1603825624, 0),

		Content: string("\x00asm\x01\x00\x00\x00\x01\x86\x01\x14`\x00\x00`\x01\u007f\x00`\x01~\x00`\x06~~~~\u007f\u007f\x01\u007f`\x05~~~~\u007f\x01\u007f`\x04~~~~\x01\u007f`\x00\x01\u007f`\x02\u007f\u007f\x00`\x03\u007f\u007f\u007f\x01\u007f`\x02\u007f\u007f\x01\u007f`\x04\u007f~~\u007f\x00`\x02\u007f~\x00`\x03~~~\x00`\x01\u007f\x01\u007f`\x06\u007f~~~~\u007f\x00`\x06\u007f~~~~~\x00`\a\u007f~~~~~~\x00`\x06\u007f~~~~|\x00`\x05\u007f~~~~\x00`\x02~~\x00\x02\xea\x02\x13\x03env\x06prints\x00\x01\x03env\x06printn\x00\x02\x03env\fdb_store_i64\x00\x03\x03env\x06printi\x00\x02\x03env\x0edb_idx64_store\x00\x04\x03env\x0fdb_idx128_store\x00\x04\x03env\x0fdb_idx256_store\x00\x03\x03env\x13db_idx_double_store\x00\x04\x03env\x18db_idx_long_double_store\x00\x04\x03env\vdb_find_i64\x00\x05\x03env\rdb_remove_i64\x00\x01\x03env\x10action_data_size\x00\x06\x03env\feosio_assert\x00\a\x03env\x06memset\x00\b\x03env\x10read_action_data\x00\t\x03env\x06memcpy\x00\b\x03env\x05abort\x00\x00\x03env\t__ashlti3\x00\n\x03env\x11eosio_assert_code\x00\v\x03\"!\x00\f\r\x01\x00\x06\t\b\r\r\x01\x01\t\t\a\a\x01\x01\x0e\x0f\x10\x11\x10\x12\x13\t\a\x13\x13\x13\x13\x13\x13\x04\x05\x01p\x01\x01\x01\x05\x03\x01\x00\x01\x06\x16\x03\u007f\x01A\x80\xc0\x00\v\u007f\x00A\x8b\xc2\x00\v\u007f\x00A\x8b\xc2\x00\v\a\t\x01\x05apply\x00\x14\n\x82+!\x04\x00\x10\x17\v\x8a\x02\x00\x10\x13 \x00 \x01Q\x04@B\x80\x80\x80\x80\xc0\x8c\xa9\xef\xf4\x00 \x02Q\x04@ \x00 \x01\x10+\x05B\x80\x80\x80\x80\x80\x80\xb8\xbd\xf2\x00 \x02Q\x04@ \x00 \x01\x10.\x05B\x80\x80\x80\x80\x80\u0e7d\xf2\x00 \x02Q\x04@ \x00 \x01\x10/\x05B\x80\x80\x80\x80\x80\x80\xa0\xbd\xf2\x00 \x02Q\x04@ \x00 \x01\x100\x05B\x80\x80\x80\x80\xc0\xf8\xa4\xbd\xf2\x00 \x02Q\x04@ \x00 \x01\x101\x05B\x80\x80\x80\x80\xe2\x93Ž\xf2\x00 \x02Q\x04@ \x00 \x01\x102\x05B\x80\x80\x80\x80\x80\x90\xa3\xea\xd3\x00 \x02Q\x04@ \x00 \x01\x103\x05 \x00B\x80\x80\x80\x80\x80\xc0\xba\x98\xd5\x00R\x04@A\x00B\x80\x80\x80\xd9ӳ\xed\x82\xef\x00\x10\x12\v\v\v\v\v\v\v\v\x05B\x80\x80\x80\x80\x80\xc0\xba\x98\xd5\x00 \x01Q\x04@B\x80\x80\x80\x80\xae\xfa\xde\xea\xa4\u007f \x02Q\x04@A\x00B\x81\x80\x80\xd9ӳ\xed\x82\xef\x00\x10\x12\v\v\vA\x00\x10$\v\x80\x01\x01\x03\u007f\x02@\x02@\x02@\x02@ \x00E\r\x00A\x00A\x00(\x02\x8c@ \x00A\x10v\"\x01j\"\x026\x02\x8c@A\x00A\x00(\x02\x84@\"\x03 \x00jA\ajAxq\"\x006\x02\x84@ \x02A\x10t \x00M\r\x01 \x01@\x00A\u007fF\r\x02\f\x03\vA\x00\x0f\vA\x00 \x02A\x01j6\x02\x8c@ \x01A\x01j@\x00A\u007fG\r\x01\vA\x00A\x9c\xc0\x00\x10\f \x03\x0f\v \x03\v\x02\x00\v6\x01\x01\u007f#\x00A\x10k\"\x00A\x006\x02\fA\x00 \x00(\x02\f(\x02\x00A\ajAxq\"\x006\x02\x84@A\x00 \x006\x02\x80@A\x00?\x006\x02\x8c@\v\x06\x00A\x90\xc0\x00\v\xf5\x01\x01\x06\u007fA\x00!\x02\x02@\x02@A\x00 \x00k\"\x03 \x00q \x00G\r\x00 \x00A\x10K\r\x01 \x01\x10\x15\x0f\v\x10\x18A\x166\x02\x00A\x00\x0f\v\x02@\x02@\x02@ \x00A\u007fj\"\x04 \x01j\x10\x15\"\x00E\r\x00 \x00 \x04 \x00j \x03q\"\x02F\r\x01 \x00A|j\"\x03(\x02\x00\"\x04A\aq\"\x01E\r\x02 \x00 \x04Axqj\"\x04Axj\"\x05(\x02\x00!\x06 \x03 \x01 \x02 \x00k\"\ar6\x02\x00 \x02A|j \x04 \x02k\"\x03 \x01r6\x02\x00 \x02Axj \x06A\aq\"\x01 \ar6\x02\x00 \x05 \x01 \x03r6\x02\x00 \x00\x10\x16\v \x02\x0f\v \x00\x0f\v \x02Axj \x00Axj(\x02\x00 \x02 \x00k\"\x00j6\x02\x00 \x02A|j \x03(\x02\x00 \x00k6\x02\x00 \x02\v3\x01\x01\u007fA\x16!\x03\x02@\x02@ \x01A\x04I\r\x00 \x01 \x02\x10\x19\"\x01E\r\x01 \x00 \x016\x02\x00A\x00!\x03\v \x03\x0f\v\x10\x18(\x02\x00\v8\x01\x02\u007f\x02@ \x00A\x01 \x00\x1b\"\x01\x10\x15\"\x00\r\x00\x03@A\x00!\x00A\x00(\x02\x98@\"\x02E\r\x01 \x02\x11\x00\x00 \x01\x10\x15\"\x00E\r\x00\v\v \x00\v\x06\x00 \x00\x10\x1b\v\x0e\x00\x02@ \x00E\r\x00 \x00\x10\x16\v\v\x06\x00 \x00\x10\x1d\vk\x01\x02\u007f#\x00A\x10k\"\x02$\x00\x02@ \x02A\fj \x01A\x04 \x01A\x04K\x1b\"\x01 \x00A\x01 \x00\x1b\"\x03\x10\x1aE\r\x00\x02@\x03@A\x00(\x02\x98@\"\x00E\r\x01 \x00\x11\x00\x00 \x02A\fj \x01 \x03\x10\x1a\r\x00\f\x02\v\v \x02A\x006\x02\f\v \x02(\x02\f!\x00 \x02A\x10j$\x00 \x00\v\b\x00 \x00 \x01\x10\x1f\v\x0e\x00\x02@ \x00E\r\x00 \x00\x10\x16\v\v\b\x00 \x00 \x01\x10!\v\x05\x00\x10\x10\x00\v\x02\x00\v^\x01\x01\u007fA\xb5\xc0\x00\x10\x00 \x01\x10\x01A\xbd\xc0\x00\x10\x00 \x02\x10\x01A\xbf\xc0\x00\x10\x00 \x04\x10\x01A\xbd\xc0\x00\x10\x00 \x03\x10\x01A\xc2\xc0\x00\x10\x00 \x02 \x01 \x03 \x04 \x05(\x02\x00\"\x06 \x05(\x02\x04 \x06k\x10\x02!\x05A\xc5\xc0\x00\x10\x00 \x05\xac\x10\x03A\xd3\xc0\x00\x10\x00\vk\x01\x02\u007f#\x00A\x10k\"\x06$\x00 \x06 \x057\x03\bA\xd5\xc0\x00\x10\x00 \x01\x10\x01A\xbd\xc0\x00\x10\x00 \x02\x10\x01A\xbf\xc0\x00\x10\x00 \x04\x10\x01A\xbd\xc0\x00\x10\x00 \x03\x10\x01A\xc2\xc0\x00\x10\x00 \x02 \x01 \x03 \x04 \x06A\bj\x10\x04!\aA\xdb\xc0\x00\x10\x00 \a\xac\x10\x03A\xd3\xc0\x00\x10\x00 \x06A\x10j$\x00\vo\x01\x02\u007f#\x00A\x10k\"\a$\x00 \a \x067\x03\b \a \x057\x03\x00A\xe7\xc0\x00\x10\x00 \x01\x10\x01A\xbd\xc0\x00\x10\x00 \x02\x10\x01A\xbf\xc0\x00\x10\x00 \x04\x10\x01A\xbd\xc0\x00\x10\x00 \x03\x10\x01A\xc2\xc0\x00\x10\x00 \x02 \x01 \x03 \x04 \a\x10\x05!\bA\xee\xc0\x00\x10\x00 \b\xac\x10\x03A\xd3\xc0\x00\x10\x00 \aA\x10j$\x00\vk\x01\x02\u007f#\x00A\x10k\"\x06$\x00 \x06 \x059\x03\bA\x8d\xc1\x00\x10\x00 \x01\x10\x01A\xbd\xc0\x00\x10\x00 \x02\x10\x01A\xbf\xc0\x00\x10\x00 \x04\x10\x01A\xbd\xc0\x00\x10\x00 \x03\x10\x01A\xc2\xc0\x00\x10\x00 \x02 \x01 \x03 \x04 \x06A\bj\x10\a!\aA\x95\xc1\x00\x10\x00 \a\xac\x10\x03A\xd3\xc0\x00\x10\x00 \x06A\x10j$\x00\vo\x01\x02\u007f#\x00A\x10k\"\a$\x00 \a \x067\x03\b \a \x057\x03\x00A\xa3\xc1\x00\x10\x00 \x01\x10\x01A\xbd\xc0\x00\x10\x00 \x02\x10\x01A\xbf\xc0\x00\x10\x00 \x04\x10\x01A\xbd\xc0\x00\x10\x00 \x03\x10\x01A\xc2\xc0\x00\x10\x00 \x02 \x01 \x03 \x04 \a\x10\b!\bA\xac\xc1\x00\x10\x00 \b\xac\x10\x03A\xd3\xc0\x00\x10\x00 \aA\x10j$\x00\vQ\x01\x01\u007fA\xbb\xc1\x00\x10\x00 \x01\x10\x01A\xbd\xc0\x00\x10\x00 \x02\x10\x01A\xbd\xc0\x00\x10\x00 \x03\x10\x01A\xbf\xc0\x00\x10\x00 \x04\x10\x01A\xc2\xc0\x00\x10\x00 \x01 \x03 \x02 \x04\x10\t\"\x05\x10\nA\xc3\xc1\x00\x10\x00 \x05\xac\x10\x03A\xd3\xc0\x00\x10\x00\v\xf0\x04\x02\x04\u007f\x04~#\x00A\xd0\x00k\"\x02!\x03 \x02$\x00\x02@\x02@\x02@\x02@\x10\v\"\x04E\r\x00 \x04A\x80\x04I\r\x01 \x04\x10\x15!\x02\f\x02\vA\x00!\x02\f\x02\v \x02 \x04A\x0fjApqk\"\x02$\x00\v \x02 \x04\x10\x0e\x1a\v \x03 \x026\x02D \x03 \x026\x02@ \x03 \x02 \x04j\"\x056\x02H \x03B\x007\x038\x02@ \x04A\aK\r\x00A\x00A\xda\xc1\x00\x10\f \x03A\xc8\x00j(\x02\x00!\x05 \x03(\x02D!\x02\v \x03A8j \x02A\b\x10\x0f\x1a \x03 \x02A\bj\"\x026\x02D \x03B\x007\x030\x02@ \x05 \x02kA\aK\r\x00A\x00A\xda\xc1\x00\x10\f \x03A\xc0\x00jA\bj(\x02\x00!\x05 \x03(\x02D!\x02\v \x03A0j \x02A\b\x10\x0f\x1a \x03 \x02A\bj\"\x026\x02D \x03B\x007\x03(\x02@ \x05 \x02kA\aK\r\x00A\x00A\xda\xc1\x00\x10\f \x03A\xc8\x00j(\x02\x00!\x05 \x03(\x02D!\x02\v \x03A(j \x02A\b\x10\x0f\x1a \x03 \x02A\bj\"\x026\x02D \x03B\x007\x03 \x02@ \x05 \x02kA\aK\r\x00A\x00A\xda\xc1\x00\x10\f \x03(\x02D!\x02\v \x03A j \x02A\b\x10\x0f\x1a \x03 \x02A\bj6\x02D \x03A\x006\x02\x18 \x03B\x007\x03\x10 \x03A\xc0\x00j \x03A\x10j\x10,\x1a \x03B\x007\x03\x00 \x03A\x006\x02\b \x03)\x03 !\x06 \x03)\x03(!\a \x03)\x030!\b \x03)\x038!\t\x02@\x02@ \x03(\x02\x14 \x03(\x02\x10k\"\x02E\r\x00 \x02A\u007fL\r\x01 \x03A\bj \x02\x10\x1b\"\x05 \x02j6\x02\x00 \x03 \x056\x02\x00 \x03 \x056\x02\x04 \x03(\x02\x14 \x03(\x02\x10\"\x04k\"\x02A\x01H\r\x00 \x05 \x04 \x02\x10\x0f\x1a \x03 \x03(\x02\x04 \x02j6\x02\x04\v \x03 \t \b \a \x06 \x03\x10%\x02@ \x03(\x02\x00\"\x02E\r\x00 \x03 \x026\x02\x04 \x02\x10\x1d\v\x02@ \x03(\x02\x10\"\x02E\r\x00 \x03 \x026\x02\x14 \x02\x10\x1d\v \x03A\xd0\x00j$\x00\x0f\v \x03\x10#\x00\v\xa1\x02\x03\x01\u007f\x01~\x05\u007f \x00(\x02\x04!\x02B\x00!\x03 \x00A\bj!\x04 \x00A\x04j!\x05A\x00!\x06\x03@\x02@ \x02 \x04(\x02\x00I\r\x00A\x00A\xd6\xc1\x00\x10\f \x05(\x02\x00!\x02\v \x02-\x00\x00!\a \x05 \x02A\x01j\"\b6\x02\x00 \x03 \aA\xff\x00q \x06A\xff\x01q\"\x02t\xad\x84!\x03 \x02A\aj!\x06 \b!\x02 \aA\x80\x01q\r\x00\v\x02@\x02@ \x01(\x02\x04\"\a \x01(\x02\x00\"\x02k\"\x05 \x03\xa7\"\x06O\r\x00 \x01 \x06 \x05k\x10- \x00A\x04j(\x02\x00!\b \x01A\x04j(\x02\x00!\a \x01(\x02\x00!\x02\f\x01\v \x05 \x06M\r\x00 \x01A\x04j \x02 \x06j\"\a6\x02\x00\v\x02@ \x00A\bj(\x02\x00 \bk \a \x02k\"\aO\r\x00A\x00A\xda\xc1\x00\x10\f \x00A\x04j(\x02\x00!\b\v \x02 \b \a\x10\x0f\x1a \x00A\x04j\"\x02 \x02(\x02\x00 \aj6\x02\x00 \x00\v\xbe\x02\x01\x06\u007f\x02@\x02@\x02@\x02@\x02@ \x00(\x02\b\"\x02 \x00(\x02\x04\"\x03k \x01O\r\x00 \x03 \x00(\x02\x00\"\x04k\"\x05 \x01j\"\x06A\u007fL\r\x02A\xff\xff\xff\xff\a!\a\x02@ \x02 \x04k\"\x02A\xfe\xff\xff\xff\x03K\r\x00 \x06 \x02A\x01t\"\x02 \x02 \x06I\x1b\"\aE\r\x02\v \a\x10\x1b!\x02\f\x03\v \x00A\x04j!\x00\x03@ \x03A\x00:\x00\x00 \x00 \x00(\x02\x00A\x01j\"\x036\x02\x00 \x01A\u007fj\"\x01\r\x00\f\x04\v\vA\x00!\aA\x00!\x02\f\x01\v \x00\x10#\x00\v \x02 \aj!\a \x03 \x01j \x04k!\x04 \x02 \x05j\"\x05!\x03\x03@ \x03A\x00:\x00\x00 \x03A\x01j!\x03 \x01A\u007fj\"\x01\r\x00\v \x02 \x04j!\x04 \x05 \x00A\x04j\"\x06(\x02\x00 \x00(\x02\x00\"\x01k\"\x03k!\x02\x02@ \x03A\x01H\r\x00 \x02 \x01 \x03\x10\x0f\x1a \x00(\x02\x00!\x01\v \x00 \x026\x02\x00 \x06 \x046\x02\x00 \x00A\bj \a6\x02\x00 \x01E\r\x00 \x01\x10\x1d\x0f\v\v\xd9\x02\x01\x04\u007f#\x00A0k\"\x02!\x03 \x02$\x00\x02@\x02@\x02@\x02@\x02@\x10\v\"\x04E\r\x00 \x04A\x80\x04I\r\x01 \x04\x10\x15!\x02\f\x02\v \x03B\x007\x03(A\x00!\x02 \x03A(j!\x05\f\x02\v \x02 \x04A\x0fjApqk\"\x02$\x00\v \x02 \x04\x10\x0e\x1a \x03B\x007\x03( \x03A(j!\x05 \x04A\aK\r\x01\vA\x00A\xda\xc1\x00\x10\f\v \x05 \x02A\b\x10\x0f\x1a \x03B\x007\x03  \x02A\bj!\x05\x02@ \x04Axq\"\x04A\bG\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A j \x05A\b\x10\x0f\x1a \x03B\x007\x03\x18 \x02A\x10j!\x05\x02@ \x04A\x10G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x18j \x05A\b\x10\x0f\x1a \x03B\x007\x03\x10 \x02A\x18j!\x05\x02@ \x04A\x18G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x10j \x05A\b\x10\x0f\x1a \x02A j!\x02\x02@ \x04A G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\bj \x02A\b\x10\x0f\x1a \x03 \x03)\x03( \x03)\x03  \x03)\x03\x18 \x03)\x03\x10 \x03)\x03\b\x10& \x03A0j$\x00\v\xde\x02\x01\x05\u007f#\x00A0k\"\x02!\x03 \x02$\x00\x02@\x02@\x02@\x02@\x02@\x10\v\"\x04E\r\x00 \x04A\x80\x04I\r\x01 \x04\x10\x15!\x02\f\x02\v \x03B\x007\x03(A\x00!\x02 \x03A(j!\x05\f\x02\v \x02 \x04A\x0fjApqk\"\x02$\x00\v \x02 \x04\x10\x0e\x1a \x03B\x007\x03( \x03A(j!\x05 \x04A\aK\r\x01\vA\x00A\xda\xc1\x00\x10\f\v \x05 \x02A\b\x10\x0f\x1a \x03B\x007\x03  \x02A\bj!\x06\x02@ \x04Axq\"\x05A\bG\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A j \x06A\b\x10\x0f\x1a \x03B\x007\x03\x18 \x02A\x10j!\x06\x02@ \x05A\x10G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x18j \x06A\b\x10\x0f\x1a \x03B\x007\x03\x10 \x02A\x18j!\x06\x02@ \x05A\x18G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x10j \x06A\b\x10\x0f\x1a \x02A j!\x02\x02@ \x04ApqA G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03 \x02A\x10\x10\x0f\x1a \x03 \x03)\x03( \x03)\x03  \x03)\x03\x18 \x03)\x03\x10 \x03)\x03\x00 \x03)\x03\b\x10' \x03A0j$\x00\v\xb2\a\x04\x05\u007f\x02~\x01\u007f\x02~#\x00A\xb0\x01k\"\x02!\x03 \x02$\x00\x02@\x02@\x02@\x02@\x02@\x10\v\"\x04E\r\x00 \x04A\x80\x04I\r\x01 \x04\x10\x15!\x05\f\x02\v \x03B\x007\x03hA\x00!\x05 \x03A\xe8\x00j!\x02\f\x02\v \x02 \x04A\x0fjApqk\"\x05$\x00\v \x05 \x04\x10\x0e\x1a \x03B\x007\x03h \x03A\xe8\x00j!\x02 \x04A\aK\r\x01\vA\x00A\xda\xc1\x00\x10\f\v \x02 \x05A\b\x10\x0f\x1a \x03B\x007\x03` \x05A\bj!\x06\x02@ \x04Axq\"\x02A\bG\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\xe0\x00j \x06A\b\x10\x0f\x1a \x03B\x007\x03X \x05A\x10j!\x06\x02@ \x02A\x10G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\xd8\x00j \x06A\b\x10\x0f\x1a \x03B\x007\x03P \x05A\x18j!\x06\x02@ \x02A\x18G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\xd0\x00j \x06A\b\x10\x0f\x1a \x03A0jA\x18jB\x007\x03\x00A\x10!\x02 \x03A0jA\x10jB\x007\x03\x00 \x03B\x007\x038 \x03B\x007\x030 \x05A j!\x05\x02@ \x04A`qA G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x90\x01j \x05A \x10\x0f\x1aB\x00!\a \x03A\xf0\x00j!\x04A\x00!\x05B\x00!\b\x02@\x03@ \x03A\x90\x01j \x05j!\x06\x02@ \x02A\x02I\r\x00 \bB\b\x86 \a \x061\x00\x00\x84\"\aB8\x88\x84!\b \x02A\u007fj!\x02 \aB\b\x86!\a \x05A\x01j\"\x05A G\r\x01\f\x02\v\x02@ \x02A\x01F\r\x00A\x00A\xdf\xc1\x00\x10\f\v \x04 \b7\x03\b \x04 \a \x061\x00\x00\x847\x03\x00A\x10!\x02 \x04A\x10j!\x04B\x00!\aB\x00!\b \x05A\x01j\"\x05A G\r\x00\v\v\x02@ \x02A\x10F\r\x00\x02@ \x02A\x02I\r\x00 \x03 \a \b \x02A\x03tAxj\x10\x11 \x03A\bj)\x03\x00!\b \x03)\x03\x00!\a\v \x04 \a7\x03\x00 \x04 \b7\x03\b\v \x03A0jA\x18j\"\x04 \x03A\xf0\x00jA\x18j\"\x02)\x03\x007\x03\x00 \x03A0jA\x10j\"\x06 \x03A\xf0\x00jA\x10j\"\x05)\x03\x007\x03\x00 \x03 \x03)\x03x7\x038 \x03 \x03)\x03p7\x030 \x03A\x10jA\x10j\"\t \x06)\x03\x007\x03\x00 \x03A\x10jA\x18j\"\x06 \x04)\x03\x007\x03\x00 \x03 \x03)\x0307\x03\x10 \x03 \x03)\x0387\x03\x18 \x03)\x03X!\a \x03)\x03P!\b \x03)\x03`!\n \x03)\x03h!\v \x02 \x06)\x03\x007\x03\x00 \x05 \t)\x03\x007\x03\x00 \x03 \x03)\x03\x187\x03x \x03 \x03)\x03\x107\x03pA\xfb\xc0\x00\x10\x00 \v\x10\x01A\xbd\xc0\x00\x10\x00 \n\x10\x01A\xbf\xc0\x00\x10\x00 \b\x10\x01A\xbd\xc0\x00\x10\x00 \a\x10\x01A\xc2\xc0\x00\x10\x00 \x03A\x90\x01jA\x18j \x02)\x03\x007\x03\x00 \x03A\x90\x01jA\x10j \x05)\x03\x007\x03\x00 \x03 \x03)\x03x7\x03\x98\x01 \x03 \x03)\x03p7\x03\x90\x01 \n \v \a \b \x03A\x90\x01jA\x02\x10\x06!\x02A\x81\xc1\x00\x10\x00 \x02\xac\x10\x03A\xd3\xc0\x00\x10\x00 \x03A\xb0\x01j$\x00\v\xd9\x02\x01\x04\u007f#\x00A0k\"\x02!\x03 \x02$\x00\x02@\x02@\x02@\x02@\x02@\x10\v\"\x04E\r\x00 \x04A\x80\x04I\r\x01 \x04\x10\x15!\x02\f\x02\v \x03B\x007\x03(A\x00!\x02 \x03A(j!\x05\f\x02\v \x02 \x04A\x0fjApqk\"\x02$\x00\v \x02 \x04\x10\x0e\x1a \x03B\x007\x03( \x03A(j!\x05 \x04A\aK\r\x01\vA\x00A\xda\xc1\x00\x10\f\v \x05 \x02A\b\x10\x0f\x1a \x03B\x007\x03  \x02A\bj!\x05\x02@ \x04Axq\"\x04A\bG\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A j \x05A\b\x10\x0f\x1a \x03B\x007\x03\x18 \x02A\x10j!\x05\x02@ \x04A\x10G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x18j \x05A\b\x10\x0f\x1a \x03B\x007\x03\x10 \x02A\x18j!\x05\x02@ \x04A\x18G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x10j \x05A\b\x10\x0f\x1a \x02A j!\x02\x02@ \x04A G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\bj \x02A\b\x10\x0f\x1a \x03 \x03)\x03( \x03)\x03  \x03)\x03\x18 \x03)\x03\x10 \x03+\x03\b\x10( \x03A0j$\x00\v\xde\x02\x01\x05\u007f#\x00A0k\"\x02!\x03 \x02$\x00\x02@\x02@\x02@\x02@\x02@\x10\v\"\x04E\r\x00 \x04A\x80\x04I\r\x01 \x04\x10\x15!\x02\f\x02\v \x03B\x007\x03(A\x00!\x02 \x03A(j!\x05\f\x02\v \x02 \x04A\x0fjApqk\"\x02$\x00\v \x02 \x04\x10\x0e\x1a \x03B\x007\x03( \x03A(j!\x05 \x04A\aK\r\x01\vA\x00A\xda\xc1\x00\x10\f\v \x05 \x02A\b\x10\x0f\x1a \x03B\x007\x03  \x02A\bj!\x06\x02@ \x04Axq\"\x05A\bG\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A j \x06A\b\x10\x0f\x1a \x03B\x007\x03\x18 \x02A\x10j!\x06\x02@ \x05A\x10G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x18j \x06A\b\x10\x0f\x1a \x03B\x007\x03\x10 \x02A\x18j!\x06\x02@ \x05A\x18G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x10j \x06A\b\x10\x0f\x1a \x02A j!\x02\x02@ \x04ApqA G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03 \x02A\x10\x10\x0f\x1a \x03 \x03)\x03( \x03)\x03  \x03)\x03\x18 \x03)\x03\x10 \x03)\x03\x00 \x03)\x03\b\x10) \x03A0j$\x00\v\xac\x02\x01\x04\u007f#\x00A k\"\x02!\x03 \x02$\x00\x02@\x02@\x02@\x02@\x02@\x10\v\"\x04E\r\x00 \x04A\x80\x04I\r\x01 \x04\x10\x15!\x02\f\x02\v \x03B\x007\x03\x18A\x00!\x02 \x03A\x18j!\x05\f\x02\v \x02 \x04A\x0fjApqk\"\x02$\x00\v \x02 \x04\x10\x0e\x1a \x03B\x007\x03\x18 \x03A\x18j!\x05 \x04A\aK\r\x01\vA\x00A\xda\xc1\x00\x10\f\v \x05 \x02A\b\x10\x0f\x1a \x03B\x007\x03\x10 \x02A\bj!\x05\x02@ \x04Axq\"\x04A\bG\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\x10j \x05A\b\x10\x0f\x1a \x03B\x007\x03\b \x02A\x10j!\x05\x02@ \x04A\x10G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03A\bj \x05A\b\x10\x0f\x1a \x03B\x007\x03\x00 \x02A\x18j!\x02\x02@ \x04A\x18G\r\x00A\x00A\xda\xc1\x00\x10\f\v \x03 \x02A\b\x10\x0f\x1a \x03 \x03)\x03\x18 \x03)\x03\x10 \x03)\x03\b \x03)\x03\x00\x10* \x03A j$\x00\v\v\x8c\x03\x16\x00A\x9c\xc0\x00\v!failed to allocate pages\x00inject \x00\x00A\xbd\xc0\x00\v\x02:\x00\x00A\xbf\xc0\x00\v\x03 <\x00\x00A\xc2\xc0\x00\v\x03>\n\x00\x00A\xc5\xc0\x00\v\x0einject resp: \x00\x00A\xd3\xc0\x00\v\x02\n\x00\x00A\xd5\xc0\x00\v\x06idxi \x00\x00A\xdb\xc0\x00\v\fidxi resp: \x00\x00A\xe7\xc0\x00\v\aidxii \x00\x00A\xee\xc0\x00\v\ridxii resp: \x00\x00A\xfb\xc0\x00\v\x06idxc \x00\x00A\x81\xc1\x00\v\fidxc resp: \x00\x00A\x8d\xc1\x00\v\bidxdbl \x00\x00A\x95\xc1\x00\v\x0eidxdbl resp: \x00\x00A\xa3\xc1\x00\v\tidxldbl \x00\x00A\xac\xc1\x00\v\x0fidxldbl resp: \x00\x00A\xbb\xc1\x00\v\bdelete \x00\x00A\xc3\xc1\x00\v\x13idxldbl resp: itr=\x00\x00A\xd6\xc1\x00\v\x04get\x00\x00A\xda\xc1\x00\v\x05read\x00\x00A\xdf\xc1\x00\v,unexpected error in fixed_bytes constructor\x00\x00A\x00\v\x04\x10!\x00\x00"),
	}

	// define dirs
	dir1 := &embedded.EmbeddedDir{
		Filename:   "",
		DirModTime: time.Unix(1603825624, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file2, // "migrator.abi"
			file3, // "migrator.wasm"

		},
	}

	// link ChildDirs
	dir1.ChildDirs = []*embedded.EmbeddedDir{}

	// register embeddedBox
	embedded.RegisterEmbeddedBox(`./code/build`, &embedded.EmbeddedBox{
		Name: `./code/build`,
		Time: time.Unix(1603825624, 0),
		Dirs: map[string]*embedded.EmbeddedDir{
			"": dir1,
		},
		Files: map[string]*embedded.EmbeddedFile{
			"migrator.abi":  file2,
			"migrator.wasm": file3,
		},
	})
}
