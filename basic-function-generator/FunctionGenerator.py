### This is a util script that generates the basic Automation scripts for a Demisto BYOI integration.
### This gives you a good starting point from which to enhance your script to format output, handle errors etc. 
### INPUTS: BYOI integration yaml file (such as those that appear in https://github.com/demisto/content/tree/master/Integrations )
### OUTPUTS: One yaml file, containing the basic automation script, for each command in the provided integration.
#consts
IMPORT_FILE_PATH = 'Yml file path'
EXPORT_FOLDER_PATH = 'output folder path'
SCRIPT_TAG_STR = '/script-'
FILE_TYPE_STR = '.yml'

SCRIPT_STRING = '''
resp = demisto.executeCommand("{pd[name]}", demisto.args())

if isError(resp[0]):
    demisto.results(resp)
else:
    data = demisto.get(resp[0], "Contents")
    if data:
        data = data if isinstance(data, list) else [data]
        data = [{{k: formatCell(row[k]) for k in row}} for row in data]
        demisto.results({{"ContentsFormat": formats["table"], "Type": entryTypes["note"], "Contents": data}} )
    else:
        demisto.results("No results.")
'''

SCRIPT_STRING_NO_PARSE = """
demisto.results(demisto.executeCommand("{pd[name]}", demisto.args()))
"""
#depends on api implementation  
FUNCTION_NAME_SEPARATOR = '-'
#FUNCTION_NAME_SEPERATOR = '_'

#imports
import yaml
import sys
import re
from collections import OrderedDict

######################### Yaml style
class quoted(str): pass

def quoted_presenter(dumper, data):
    return dumper.represent_scalar('tag:yaml.org,2002:str', data, style='"')
yaml.add_representer(quoted, quoted_presenter)

class literal(str): pass

def literal_presenter(dumper, data):
    return dumper.represent_scalar('tag:yaml.org,2002:str', data, style='|')
yaml.add_representer(literal, literal_presenter)

def ordered_dict_presenter(dumper, data):
    return dumper.represent_dict(data.items())
yaml.add_representer(OrderedDict, ordered_dict_presenter)

######################### main
def GenerateScripts(yamlData, outputFolderPath):
	"""	Generate a script from each command in the original Yaml file

	:param yamlData: a dicitonary that's represent a yaml
	:param outputFolderPath: path in which the scripts will be generated to

	"""
	commands = yamlData['script']['commands']

	for command in commands:
		newScript = OrderedDict()

		newScript['commonfields'] = {}
		newScript['commonfields']['id'] = 'TestScript'
		newScript['commonfields']['version'] = -1

		newScript['name'] = ''
		newScript['script'] = ''	
		newScript['type'] = 'python'
		newScript['tags'] = None
		newScript['comment'] = None
		newScript['system'] = False
		newScript['args'] = None
		newScript['scripttarget'] = 0
		newScript['timeout'] = '0s'
		newScript['dependson'] = None

		#convert name to CamelCase
		newScript['name'] = ''.join(map(lambda(x) : x[0].upper() + x[1:], command['name'].split(FUNCTION_NAME_SEPARATOR)))

		newScript['commonfields']['id'] = newScript['name']
		newScript['args'] = command['arguments']
		newScript['comment'] = command['description']

		newScript['tags'] = [yamlData['commonfields']['id']]
		newScript['dependson'] = {'must' : [command['name']]}

		paramDict = {}
		paramDict['name'] = command['name']

		removeSpecialChars(newScript)

		#pd - the dicitonary to be used in the formated string
		script = SCRIPT_STRING.format(pd = paramDict)
		newScript['script'] = literal(script)

		outputFile = open(outputFolderPath + SCRIPT_TAG_STR + newScript['name'] + FILE_TYPE_STR, 'w')
		yaml.dump(newScript, outputFile, indent=2, default_flow_style=False, line_break='\r\n')

		outputFile.close()

def removeSpecialChars(data):
#striping special chars recursivly in dictionary

	if type(data) is str:
		return data.translate(None, '\t\n\r\\')
	elif isinstance(data, dict):
		for key in data.keys():
			data[key] = removeSpecialChars(data[key])
		return data
	elif isinstance(data, list):
		return map(removeSpecialChars, data)
	else:
		return data

def main(argv):

	if len(argv) == 3:
		yamlFilePath = argv[1]
		outputFolderPath = argv[2]
	else:
		yamlFilePath = IMPORT_FILE_PATH
		outputFolderPath = EXPORT_FOLDER_PATH

	yamlFile  = open(yamlFilePath)
	dataMap = yaml.safe_load(yamlFile )
	yamlFile.close()

	GenerateScripts(dataMap, outputFolderPath)

main(sys.argv)

