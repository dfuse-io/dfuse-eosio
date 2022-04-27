const process = require('process')
const fs = require('fs')
const fsp = require('fs').promises;
const execSync = require('child_process').execSync;

function help() {
    console.log(
"Usage:\n\
     node consolidate_abi contracts_dir migrator.abi_path"
    );
    return 1;
};

function findAbiFiles(dir) {
    const result = execSync(`find ${dir} -name "*.abi"`,  { encoding: 'utf-8' });
    const abis = result.split("\n");
    var filteredAbis = [];
    for (const abi of abis) {
        var include = true;
        include = include && abi.length > 0;
        include = include && !abi.includes("eosio.bios");
        if(include)
            filteredAbis.push(abi);
    }
    return filteredAbis;
}

async function readAbi(abiFilePath) {
    const data = await fsp.readFile(abiFilePath, 'utf8');
    var abi = JSON.parse(data);
    if(!abi) {
        console.error("Cannot parse the migrator.abi");
        throw "Cannot read the abi file: " + abiFilePath;
    }
    return abi;
}

function compareArrays(arr1, arr2) {
    if(arr1.length != arr2.length)
        return false;
    for(var i = 0; i < arr1.length; ++i) {
        if(arr1[i] !== arr2[i])
            return false;
    }
    return true;
}

function purifyTypeName(tpName) {
    return tpName.replace(/[\[\]\?\$]/g, "");
}

function buildDefinitionChain(abi, definitionName, definitions) {
    definitionName = purifyTypeName(definitionName);

    for(const tp of abi.types) {
        if (tp.new_type_name === definitionName) {
            definitions.push({definitionName: definitionName, key: "types", value: tp});
            return buildDefinitionChain(abi, tp.type, definitions)
        }
    }

    for(const st of abi.structs) {
        if(st.name === definitionName) {
            definitions.push({definitionName: definitionName, key: "structs", value: st});
            var defs = buildDefinitionChain(abi, st.base, definitions);
            for(const tp of st.fields) {
                defs = buildDefinitionChain(abi, tp.type, defs);
            }
            return defs;
        }
    }

    for(const vr of abi.variants) {
        if(vr.name === definitionName) {
            definitions.push({definitionName: definitionName, key: "variants", value: vr});
            var defs = definitions;
            for(const tp of vr.types) {
                defs = buildDefinitionChain(abi, tp, defs);
            }
            return defs;
        }
    }

    return definitions;
}

function mergeDefinitionChain(abi, definitionChains) {
    for(const def of definitionChains) {
        const existing = abi[def.key].find((v, i)=>{
            if(def.key == "types")
                return v.new_type_name == def.definitionName;
            return v.name == def.definitionName;
        });
        if(existing)
            if(JSON.stringify(existing) !== JSON.stringify(def.value))
                throw "Existing: " + JSON.stringify(existing, null, 2);
        if(!existing)
            abi[def.key].push(def.value);
    }
}

function mergeTables(srcAbi, dstAbi) {
    for(const tbl of srcAbi.tables) {
        if(dstAbi.tables.find((value, index)=>{
            return tbl.name === value.name;
        })) throw("Table: found a duplicate: " + tbl.name);

        dstAbi.tables.push(tbl);

        var defs = [];
        buildDefinitionChain(srcAbi, tbl.type, defs);
        mergeDefinitionChain(dstAbi, defs);
    }
}


async function main() {
    if (process.argv.length < 4)
        return help();

    const contractsDirPath = process.argv[2];
    const migratorAbiPath = process.argv[3];

    if(!fs.existsSync(contractsDirPath) || !fs.existsSync(migratorAbiPath))
        return help();

    var migratorAbi = await readAbi(migratorAbiPath);

    if(!migratorAbi.hasOwnProperty("types"))
        migratorAbi["types"] = [];
    if(!migratorAbi.hasOwnProperty("structs"))
        migratorAbi["structs"] = [];
    if(!migratorAbi.hasOwnProperty("tables"))
        migratorAbi["tables"] = [];

    const contractAbis = findAbiFiles(contractsDirPath);

    for(const abiPath of contractAbis) {
        const contractAbi = await readAbi(abiPath);
        if(contractAbi.tables.length == 0)
            continue;
        mergeTables(contractAbi, migratorAbi);
    }

    await fsp.writeFile(migratorAbiPath+"_modified", JSON.stringify(migratorAbi, null, 2));
}

if (require.main === module) {
    main();
}


