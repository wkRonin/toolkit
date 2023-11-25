/*
 *    Copyright 2023 wkRonin
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package zapx

func String(key, val string) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Strings(key string, val []string) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Error(val error) Field {
	return Field{
		Key:   "error",
		Value: val,
	}
}

func Bool(key string, val bool) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Any(key string, val any) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Int64(key string, val int64) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Int64s(key string, val []int64) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Int32(key string, val int32) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}
